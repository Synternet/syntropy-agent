package saas

import (
	"context"
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

	"github.com/SyntropyNet/syntropy-agent-go/internal/config"
	"github.com/SyntropyNet/syntropy-agent-go/internal/logger"
	"github.com/SyntropyNet/syntropy-agent-go/pkg/common"
	"github.com/SyntropyNet/syntropy-agent-go/pkg/scontext"
	"github.com/gorilla/websocket"
)

const pkgName = "Saas Controller. "

var ErrNotRunning = errors.New("controller is not running.")

type CloudController struct {
	sync.Mutex // this lock makes Write thread safe
	log        *logger.Logger
	ws         *websocket.Conn
	url        string
	token      string
	version    string

	ctx scontext.StartStopContext
}

// New allocates instance of Software-As-A-Service
// (aka WSS) controller
func New(ctx context.Context) (common.Controller, error) {
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
		ctx:     scontext.New(ctx),
	}

	_, err = cc.ctx.CreateContext()
	if err != nil {
		return nil, err
	}

	// Create new local logger for controller events
	// I am using configured DebugLevel here, but actually
	// only Errors and Warnings should be logged on this logger.
	cc.log = logger.New(nil, config.GetDebugLevel(), os.Stdout)

	err = cc.connect()
	if err != nil {
		return nil, err
	}

	return &cc, nil
}

func (cc *CloudController) connect() error {
	// There may be a reader that may try to connect and writing may start during connection.
	cc.Lock()
	defer cc.Unlock()

	url := url.URL{Scheme: "wss", Host: cc.url, Path: "/"}
	headers := http.Header(make(map[string][]string))

	// Without these headers connection will be ignored silently
	headers.Set("authorization", cc.token)
	headers.Set("x-deviceid", config.GetDeviceID())
	headers.Set("x-deviceip", config.GetPublicIp())
	headers.Set("x-devicename", config.GetAgentName())
	headers.Set("x-devicestatus", "OK")
	headers.Set("x-agenttype", "Linux")
	headers.Set("x-agentversion", cc.version)

	// TODO: Implement exponential backoff with jitter
	cc.ws = nil
	for {
		select {
		case <-cc.ctx.Context().Done():
			return io.EOF
		default:
		}

		ws, resp, err := websocket.DefaultDialer.Dial(url.String(), headers)
		if err != nil {
			var httpCode int
			if resp != nil {
				httpCode = resp.StatusCode
			}
			cc.log.Error().Printf("%s ConnectionError: %s (HTTP: %d)\n", pkgName, err.Error(), httpCode)
			// Add some randomised sleep, so if controller was down
			// the reconnecting agents could DDOS the controller
			delay := time.Millisecond*100 + time.Duration(rand.Int31n(10000))*time.Millisecond
			cc.log.Warning().Println(pkgName, "Reconnecting in ", delay)
			time.Sleep(delay)
			continue
		}

		cc.ws = ws

		break
	}

	return nil
}

func (cc *CloudController) Recv() ([]byte, error) {
	if cc.ws == nil {
		cc.log.Warning().Println(pkgName, "Websocket connection to the Controller is missing.")
		return nil, ErrNotRunning
	}

	// In this application we have only one reader, so no need to lock here
	for {
		msgtype, msg, err := cc.ws.ReadMessage()

		select {
		case <-cc.ctx.Context().Done():
			return nil, io.EOF
		default:
		}

		if err == nil {
			// successfully received message
			if msgtype != websocket.TextMessage {
				cc.log.Warning().Println(pkgName, "Received unexpected message type ", msgtype)
			}
			return msg, nil
		}

		// reconnect and continue receiving
		// NOTE: connect is blocking and will block untill a connection is established
		if err := cc.connect(); err != nil {
			return nil, err
		}
	}
}

func (cc *CloudController) Write(b []byte) (n int, err error) {
	/*
		gorilla/websocket concurency:
			Connections support one concurrent reader and one concurrent writer.
			Applications are responsible for ensuring that no more than one goroutine calls the write methods
	*/
	cc.Lock()
	defer cc.Unlock()

	if cc.ws == nil {
		cc.log.Warning().Println(pkgName, "Websocket connection to the Controller is missing.")
		return 0, ErrNotRunning
	}

	err = cc.ws.WriteMessage(websocket.TextMessage, b)
	if err != nil {
		cc.log.Error().Println(pkgName, "Send error: ", err)
	} else {
		n = len(b)
	}
	return n, err
}

// Close closes websocket connection to saas backend
func (cc *CloudController) Close() error {
	cc.Lock()
	defer cc.Unlock()

	cc.ctx.CancelContext()

	if cc.ws == nil {
		return nil
	}

	// Cleanly close the connection by sending a close message and then
	// waiting (with timeout) for the server to close the connection.
	err := cc.ws.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	if err != nil {
		cc.log.Error().Println(pkgName, "connection close error: ", err)
	}

	cc.ws.Close()
	cc.ws = nil
	return nil
}

// Compile time sanity test
var _ common.Controller = &CloudController{}
