package saas

import (
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/SyntropyNet/syntropy-agent/controller"
	"github.com/SyntropyNet/syntropy-agent/internal/config"
	"github.com/SyntropyNet/syntropy-agent/internal/logger"
	"github.com/SyntropyNet/syntropy-agent/pkg/pubip"
	"github.com/SyntropyNet/syntropy-agent/pkg/state"
	"github.com/gorilla/websocket"
)

const pkgName = "Saas Controller. "
const reconnectDelay = 10000 // 10 seconds (in milliseconds)
const (
	// State machine constants
	initialised = iota
	stopped
	connecting
	running
)

var ErrNotRunning = errors.New("controller is not running")

type CloudController struct {
	// this lock makes Write thread safe
	// Use it only to protect Write calls.
	sync.Mutex
	// I like the idea of being non blocking.
	// If WSS has lost connection - all other services still are working.
	// If they try to send on bad connection - Write will exit and allow services to continue working
	// When connection is restored - services will send the newest data they have.
	// And nobody cares about old data, that was not sent during the time the connection was lost.
	state.StateMachine
	// Cloud controller nees a sepparate looger.
	// All other packages print logs to stdout (or a file maybe)
	// And also send a copy of logs to controller.
	// However the controller cannot send its logs to controller, because of 3 reasons:
	//  1. If everything is well - nobody cares what controller is logging.
	//  2. If there are a problem - it will not be logged to cloud controller.
	//  3. Logging from cloud controller may result in dead-loop (yes we may make a WAR, but its not worf it)
	// So because of all these reasons (saas) cloud controller has its own logger, logging only locally.
	// NOTE:  only Errors and Warnings should be logged on this logger  :NOTE
	log *logger.Logger
	// gorilla/websocket Connection
	ws *websocket.Conn
	// Info fields to send to cloud controller
	url     string
	token   string
	version string
}

// New allocates instance of Software-As-A-Service
// (aka WSS) controller
func New() (controller.Controller, error) {
	// validate URL early. No need to keep busy trying on invalid URLs
	url := config.GetCloudURL()
	_, err := net.LookupIP(url)
	if err != nil {
		return nil, err
	}
	if config.GetAgentToken() == "" {
		return nil, fmt.Errorf("SYNTROPY_AGENT_TOKEN is not set")
	}

	// Note: config package returns already validated values and no need to validate them here
	cc := CloudController{
		url:     url,
		token:   config.GetAgentToken(),
		version: config.GetVersion(),
	}
	cc.SetState(initialised)

	// Create new local logger for controller events
	// I am using configured DebugLevel here, but actually
	// only Errors and Warnings should be logged on this logger.
	cc.log = logger.New(nil, config.GetDebugLevel(), os.Stdout)

	return &cc, nil
}

func (cc *CloudController) Open() error {
	state := cc.GetState()
	if state != initialised {
		return fmt.Errorf("unexpected controller state %d", state)
	}

	cc.log.Info().Println(pkgName, "Connecting...")

	return cc.connect()
}

func (cc *CloudController) connect() (err error) {
	cc.SetState(connecting)
	url := url.URL{Scheme: "wss", Host: cc.url, Path: "/"}
	headers := http.Header(make(map[string][]string))

	// Without these headers connection will be ignored silently
	headers.Set("authorization", cc.token)
	headers.Set("x-deviceid", config.GetDeviceID())
	headers.Set("x-deviceip", pubip.GetPublicIp().String())
	headers.Set("x-devicename", config.GetAgentName())
	headers.Set("x-devicestatus", "OK")
	headers.Set("x-agenttype", "Linux")
	headers.Set("x-agentversion", cc.version)

	for {
		var resp *http.Response
		var httpCode int
		cc.ws, resp, err = websocket.DefaultDialer.Dial(url.String(), headers)
		if err != nil {
			if resp != nil {
				httpCode = resp.StatusCode
			}
			cc.log.Error().Printf("%s ConnectionError: %s (HTTP: %d)\n", pkgName, err.Error(), httpCode)
			// Add some randomised sleep, so if controller was down
			// the reconnecting agents could DDOS the controller
			delay := time.Duration(rand.Int31n(reconnectDelay)) * time.Millisecond
			cc.log.Warning().Println(pkgName, "Reconnecting in ", delay)
			time.Sleep(delay)
			continue
		}

		cc.SetState(running)
		break
	}

	return nil
}

func (cc *CloudController) Recv() ([]byte, error) {
	if cc.GetState() == stopped {
		return nil, ErrNotRunning
	}

	// In this application we have only one reader, so no need to lock here

	for {
		msgtype, msg, err := cc.ws.ReadMessage()

		switch {
		case err == nil:
			// successfully received message
			if msgtype != websocket.TextMessage {
				cc.log.Warning().Println(pkgName, "Received unexpected message type ", msgtype)
			}
			return msg, nil

		case cc.GetState() == stopped:
			// The connection is closed - simulate EOF
			return nil, io.EOF
		}

		// reconnect and continue receiving
		// NOTE: connect is blocking and will block untill a connection is established
		cc.connect()
	}
}

func (cc *CloudController) Write(b []byte) (n int, err error) {
	controllerState := cc.GetState()
	if controllerState != running {
		if controllerState != stopped {
			cc.log.Warning().Println(pkgName, "Controller is not running. Current state: ", controllerState)
		}
		return 0, ErrNotRunning
	}

	/*
		gorilla/websocket concurency:
			Connections support one concurrent reader and one concurrent writer.
			Applications are responsible for ensuring that no more than one goroutine calls the write methods
	*/
	cc.Lock()
	defer cc.Unlock()

	err = cc.ws.WriteMessage(websocket.TextMessage, b)
	if err != nil {
		cc.log.Error().Println(pkgName, "Send error: ", err)
	} else {
		n = len(b)
	}
	return n, err
}

// closes websocket connection to saas backend
func (cc *CloudController) close(terminate bool) error {
	if cc.GetState() == stopped {
		// cannot close already closed connection
		return ErrNotRunning
	}
	if terminate {
		cc.SetState(stopped)
	}

	//	gorilla/websocket concurency:
	//		Connections support one concurrent reader and one concurrent writer.
	//		Applications are responsible for ensuring that no more than one goroutine calls the write methods
	cc.Lock()
	defer cc.Unlock()

	// Cleanly close the connection by sending a close message and then
	// waiting (with timeout) for the server to close the connection.
	err := cc.ws.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	if err != nil {
		cc.log.Error().Println(pkgName, "connection close error: ", err)
	}

	cc.ws.Close()
	return nil
}

// Close closes websocket connection to saas backend
func (cc *CloudController) Close() error {
	return cc.close(true)
}

// Reconnect closes websocket connection to saas backend
// But does not change state - so this should result in reconnecting from Recv funtion
func (cc *CloudController) Reconnect() error {
	return cc.close(false)
}
