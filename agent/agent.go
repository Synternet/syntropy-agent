package agent

import (
	"log"
	"sync/atomic"

	"github.com/SyntropyNet/syntropy-agent-go/controller"
	"github.com/SyntropyNet/syntropy-agent-go/controller/saas"
	"github.com/SyntropyNet/syntropy-agent-go/wireguard"
	"golang.zx2c4.com/wireguard/wgctrl"
)

type Agent struct {
	running    uint32
	controller controller.Controller
	msgChanRx  chan []byte
	msgChanTx  chan []byte

	wg *wgctrl.Client

	commands map[string]func(a *Agent, req []byte) ([]byte, error)
}

// NewAgent allocates instance of agent struct
// Parses shell environment and setups internal variables
func NewAgent() (*Agent, error) {
	var err error
	agent := new(Agent)

	agent.controller, err = saas.NewCloudController()
	if err != nil {
		log.Println("Error creating cloud controller")
		return nil, err
	}

	agent.wg, err = wgctrl.New()
	if err != nil {
		log.Println("Error creating wgctrl client")
		return nil, err
	}

	agent.msgChanRx = make(chan []byte)
	agent.msgChanTx = make(chan []byte)

	agent.commands = make(map[string]func(a *Agent, req []byte) ([]byte, error))
	agent.commands["AUTO_PING"] = autoPing
	agent.commands["GET_INFO"] = getInfo
	agent.commands["CONFIG_INFO"] = configInfo

	return agent, nil
}

func (agent *Agent) messageHadler() {
	var err error

	// Mark as not running on exit
	defer atomic.StoreUint32(&agent.running, 0)

	for {
		raw, ok := <-agent.msgChanRx
		// Stop runner if the channel is closed
		if !ok {
			return
		}

		err = agent.processCommand(raw)
		if err != nil {
			log.Println(err)
			continue
		}
	}
}

func (agent *Agent) Transmit(msg []byte) {
	if atomic.LoadUint32(&agent.running) == 0 {
		log.Println("Trying to transmit on stopped agent instance")
		return
	}

	log.Println("Sending: ", string(msg))
	agent.msgChanTx <- msg
}

// Loop is main loop of SyntropyStack agent
func (agent *Agent) Loop() {
	if !atomic.CompareAndSwapUint32(&agent.running, 0, 1) {
		log.Println("Agent instance is already running")
		return
	}

	go agent.messageHadler()
	go agent.controller.Start(agent.msgChanRx, agent.msgChanTx)
}

// Stop closes connections to controller and stops all runners
func (agent *Agent) Stop() {
	if !atomic.CompareAndSwapUint32(&agent.running, 1, 0) {
		log.Println("Agent instance is not running")
		return
	}

	close(agent.msgChanTx)
	agent.controller.Stop()
	close(agent.msgChanRx)

	agent.wg.Close()
	wireguard.DestroyAllInterfaces()
}
