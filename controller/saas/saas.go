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

var heartbeatAcceptable = 45 * time.Second
var heartbeatCheckPerion = heartbeatAcceptable / 3

const (
	// State machine constants
	stopped = iota
	initialised
	connecting
	running
	disconnected
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
	// the last time received ping or pong control message
	lastHeartbeat     time.Time
	connectionIsAlive bool
	healthTimer       *time.Timer
	// Info fields to send to cloud controller
	url     string
	token   string
	version string
	// buffered channel in order not to delay sender
	messageQueue chan []byte
	// bufferLimit configures when start discarding messages when cannot send to controller
	queueLimit int
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

	if wssTimeout := config.GetWssTimeout(); wssTimeout > 0 {
		heartbeatAcceptable = time.Duration(wssTimeout) * time.Second
		heartbeatCheckPerion = heartbeatAcceptable / 3
	}

	// Note: config package returns already validated values and no need to validate them here
	cc := CloudController{
		url:     url,
		token:   config.GetAgentToken(),
		version: config.GetVersion(),
	}
	cc.SetState(initialised)

	// Prepeare health check timer
	cc.healthTimer = time.AfterFunc(heartbeatCheckPerion, cc.healthcheck)
	cc.healthTimer.Stop()

	// allocate buffered channel
	// need to experiment in different situation with values
	// Lets start with 20 messages buffer and try not to fill it more than 80%
	cc.messageQueue = make(chan []byte, 20)
	cc.queueLimit = 4

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

	// schedule the sender loop
	go cc.sendLoop()

	cc.log.Info().Println(pkgName, "Connecting...")
	cc.log.Info().Println(pkgName, "WebSocket timeout:", heartbeatAcceptable,
		"  Check period:", heartbeatCheckPerion)

	return cc.connect()
}

func (cc *CloudController) connect() (err error) {
	cc.SetState(connecting)
	cc.healthTimer.Stop()

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

		cc.log.Info().Println(pkgName, "Connected to controller", cc.ws.RemoteAddr())
		// Set ping/pong callbacks for link health monitoring
		cc.ws.SetPingHandler(cc.pingHandler)
		cc.ws.SetPongHandler(cc.pongHandler)
		// we are just connected - update the heartbeat
		cc.heartbeat()
		// start heartbeat watchdog
		cc.healthTimer.Reset(heartbeatCheckPerion)

		cc.SetState(running)
		break
	}

	return nil
}

// The point of wss connection health check is as follows:
// if server keeps pinging me - nothing's wrong and no need for active probing
// if have not received peers for some time period - try pinging other side myself
// if server is not sending ping neither responding to my pings - the connection is lost
// In that case terminate the connection and it will be restarted in Recv() function
func (cc *CloudController) healthcheck() {
	now := time.Now()
	if !cc.connectionIsAlive {
		// Connection is not sending ping or pong messages
		// Simply terminate it and reconnection will happen in Recv
		cc.log.Warning().Println(pkgName, "connection ping-pong health check failed")
		cc.ws.Close()
		return
	} else if diff := now.Sub(cc.lastHeartbeat); diff > heartbeatAcceptable {
		// Have not heard connection for a long time
		// Try pinging other side myself
		cc.connectionIsAlive = false

		cc.log.Info().Println(pkgName, "no ping from server for", diff)
		cc.Lock()
		err := cc.ws.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(time.Second))
		cc.Unlock()

		if err != nil {
			// Ping send failed - no need to wait - reconnect
			cc.ws.Close()
		}
	}
	// Schedule next watchdog check
	cc.healthTimer.Reset(heartbeatCheckPerion)
}

// updates last hearbeat time
func (cc *CloudController) heartbeat() {
	cc.connectionIsAlive = true
	cc.lastHeartbeat = time.Now()
}

func (cc *CloudController) pingHandler(message string) error {
	cc.heartbeat()

	cc.Lock()
	err := cc.ws.WriteControl(websocket.PongMessage, []byte(message), time.Now().Add(time.Second))
	cc.Unlock()
	if err == websocket.ErrCloseSent {
		return nil
	} else if e, ok := err.(net.Error); ok && e.Temporary() {
		return nil
	}
	return err
}

func (cc *CloudController) pongHandler(string) error {
	cc.heartbeat()
	return nil
}

func (cc *CloudController) sendLoop() {
	// gorilla/websocket concurency:
	// 	Connections support one concurrent reader and one concurrent writer.
	// 	Applications are responsible for ensuring that no more than one goroutine calls the write methods
	// In this application there are 2 senders:
	// this function and CloudController.close, thus lock protection is needed

	// process all messages until channel is closed
	for msg := range cc.messageQueue {
		// retry loop in case of reconnection
		for retry := true; retry; {
			// Default is retry one.
			// Note: do not put this in the for loop 3rd statement
			// it is processed then after the loop
			retry = false

			// Respect controller state machine and act accordingly
			controllerState := cc.GetState()
			switch controllerState {
			case stopped:
				// controller is stopped already. Discard all messages and will exit on closed channel
				cc.log.Debug().Println(pkgName, "Controller is stopped. Discarding remaining messages.")

			case initialised:
				// This state should never happen. Print error and discard message
				cc.log.Error().Println(pkgName, "Unexpected controller state (initialised) during runtime !")

			case connecting:
				// Controller is reconnecting
				// If channel is not almost full - try some delay and retry
				// If channel is almost full - sadly we need to discard some messages
				// TODO: think about smart messages discarding. Packet investigation or priority
				if cap(cc.messageQueue)-len(cc.messageQueue) < cc.queueLimit {
					cc.log.Warning().Println(pkgName, "send queue almost full. Discarding message")
					cc.log.Debug().Println(pkgName, "Discarded packet:  XX", string(msg), "XX")
				} else {
					retry = true
					time.Sleep(100 * time.Millisecond)
				}

			case running:
				// Expected state. Send the message
				cc.Lock()
				err := cc.ws.WriteMessage(websocket.TextMessage, msg)
				cc.Unlock()
				if err != nil {
					cc.log.Error().Println(pkgName, "Send error: ", err)
				}

			default:
				// Unsupported state? Print warning and discard message
				cc.log.Warning().Println(pkgName, "Unsupported controller state", controllerState)
			}
		}
	}
}

func (cc *CloudController) Recv() ([]byte, error) {
	if cc.GetState() == stopped {
		return nil, ErrNotRunning
	}

	// gorilla/websocket concurency:
	// 	Connections support one concurrent reader and one concurrent writer.
	// 	Applications are responsible for ensuring that no more than one goroutine calls the write methods
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
		cc.log.Info().Println(pkgName, "websocket connection was lost. Reconnecting.")
		cc.connect()
	}
}

func (cc *CloudController) Write(b []byte) (n int, err error) {
	if cc.GetState() == stopped {
		cc.log.Warning().Println(pkgName, "Controller is not running.")
		return 0, ErrNotRunning
	}

	cc.messageQueue <- b
	return len(b), nil
}

// closes websocket connection to saas backend
func (cc *CloudController) close(terminate bool) error {
	if cc.GetState() == stopped {
		// cannot close already closed connection
		return ErrNotRunning
	}
	if terminate {
		cc.SetState(stopped)
		close(cc.messageQueue)
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
	cc.healthTimer.Stop()
	return cc.close(true)
}

// Reconnect closes websocket connection to saas backend
// But does not change state - so this should result in reconnecting from Recv funtion
func (cc *CloudController) Reconnect() error {
	return cc.close(false)
}
