package saas

import (
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/SyntropyNet/syntropy-agent-go/config"
	"github.com/SyntropyNet/syntropy-agent-go/controller"
	"github.com/gorilla/websocket"
)

type CloudController struct {
	url     string
	token   string
	version string

	quit chan int

	ws *websocket.Conn
}

// NewAgent allocates instance of agent struct
// Parses shell environment and setups internal variables
func NewCloudController() (controller.Controller, error) {
	// Note: config package returns already validated values and no need to validate them here
	cc := CloudController{
		url:     config.GetCloudURL(),
		token:   config.GetAgentToken(),
		version: config.GetVersion(),
	}

	err := cc.createWebsocketConnection()
	if err != nil {
		return nil, err
	}

	cc.quit = make(chan int)

	return &cc, nil
}

func (cc *CloudController) createWebsocketConnection() (err error) {
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

	return nil
}

// Start is main loop of SyntropyStack agent
func (cc *CloudController) Start(rx, tx chan []byte) {
	defer close(cc.quit)

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

// Stop closes websocket connection
func (cc *CloudController) Stop() {
	// Cleanly close the connection by sending a close message and then
	// waiting (with timeout) for the server to close the connection.
	err := cc.ws.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	if err != nil {
		log.Println("write close:", err)
		return
	}

	select {
	case <-cc.quit:
	case <-time.After(time.Second):
	}

	cc.ws.Close()
}
