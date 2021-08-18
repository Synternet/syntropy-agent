package saas

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"

	"github.com/SyntropyNet/syntropy-agent-go/config"
	"github.com/SyntropyNet/syntropy-agent-go/controller"
	"github.com/gorilla/websocket"
)

const (
	stopped = iota
	connecting
	running
)

type CloudController struct {
	sync.Mutex
	state   uint32 // atomic state: 1 running, 0 closed
	ws      *websocket.Conn
	reader  io.Reader
	url     string
	token   string
	version string
}

// NewController allocates instance of Software-As-A-Service
// (aka WSS) controller
func NewController() (controller.Controller, error) {
	// Note: config package returns already validated values and no need to validate them here
	cc := CloudController{
		url:     config.GetCloudURL(),
		token:   config.GetAgentToken(),
		version: config.GetVersion(),
		state:   stopped,
	}

	err := cc.connect()
	if err != nil {
		return nil, err
	}

	return &cc, nil
}

func (cc *CloudController) connect() (err error) {
	// not yet atomic.StoreUint32(&cc.state, connecting)
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

	var resp *http.Response
	var httpCode int
	cc.ws, resp, err = websocket.DefaultDialer.Dial(url.String(), headers)
	if err != nil {
		if resp != nil {
			httpCode = resp.StatusCode
		}
		log.Printf("WSS dialer error: %s (HTTP: %d)\n", err.Error(), httpCode)
		return err
	}
	_, cc.reader, err = cc.ws.NextReader()
	if err != nil {
		return err
	}
	atomic.StoreUint32(&cc.state, running)

	return nil
}

func (cc *CloudController) Recv() ([]byte, error) {
	// In this application we have only one reader, so no need to lock here

	for {
		msgtype, msg, err := cc.ws.ReadMessage()

		switch {
		case err == nil:
			// successfully received message
			log.Println("Message type: ", msgtype)
			return msg, nil

		case atomic.LoadUint32(&cc.state) == stopped:
			// The connection is closed - simulate EOF
			log.Println("Saas connection is closed")
			return nil, io.EOF
		}

		log.Println("SaaS agent error: ", err, ". Reconnecting...")
		cc.connect() // reconnect and continue receiving
	}
}

func (cc *CloudController) Write(b []byte) (n int, err error) {
	if atomic.LoadUint32(&cc.state) == stopped {
		return 0, fmt.Errorf("controller is not running")
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
		log.Println("Websocket write error:", err)
	} else {
		n = len(b)
	}
	return n, err
}

// Start is main loop of SyntropyStack agent
func (cc *CloudController) Start(rx, tx chan []byte) {

	// Receiver goroutine
	go func() {
		for {
			_, message, err := cc.ws.ReadMessage()
			if err != nil {
				log.Println("Websocket read error:", err)
				return
			}
			rx <- message
		}
	}()

	// Sender goroutine
	go func() {
		for {
			message, ok := <-tx
			if !ok {
				return // terminate, because channel is closed
			}
			err := cc.ws.WriteMessage(websocket.TextMessage, message)
			if err != nil {
				log.Println("Websocket write error:", err)
				return
			}
		}
	}()

}

// Close closes websocket connection to saas backend
func (cc *CloudController) Close() error {
	state := atomic.LoadUint32(&cc.state)
	if state == stopped {
		// cannot close already closed connection
		return fmt.Errorf("controller already closed")
	}

	// Cleanly close the connection by sending a close message and then
	// waiting (with timeout) for the server to close the connection.
	err := cc.ws.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	if err != nil {
		log.Println("write close:", err)
	}
	atomic.StoreUint32(&cc.state, stopped)

	cc.ws.Close()
	return nil
}
