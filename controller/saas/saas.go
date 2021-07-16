package saas

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

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
func NewCloudController(version string) (controller.Controller, error) {
	cc := CloudController{
		url:     os.Getenv("SYNTROPY_CONTROLLER_URL"),
		token:   os.Getenv("SYNTROPY_AGENT_TOKEN"),
		version: version,
	}

	if cc.token == "" {
		return nil, fmt.Errorf("SYNTROPY_AGENT_TOKEN is not set")
	}

	if cc.url == "" {
		cc.url = "controller-prod-platform-agents.syntropystack.com"
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
	headers.Set("x-deviceid", generateDeviceId())
	headers.Set("x-deviceip", getPublicIp())
	headers.Set("x-devicename", getAgentName())
	headers.Set("x-devicestatus", "OK")
	headers.Set("x-agenttype", "Linux")
	headers.Set("x-agentversion", cc.version)

	cc.ws, _, err = websocket.DefaultDialer.Dial(url.String(), headers)
	if err != nil {
		return err
	}

	return nil
}

// Start is main loop of SyntropyStack agent
func (cc *CloudController) Start() {
	defer close(cc.quit)

	for {
		mtype, message, err := cc.ws.ReadMessage()
		if err != nil {
			log.Println("read error:", err)
			return
		}
		log.Printf("recv: [%d] %s", mtype, message)
	}
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
