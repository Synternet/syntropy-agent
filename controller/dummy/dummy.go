package dummy

import (
	"fmt"
	"io/ioutil"
	"log"
	"strings"
	"time"

	"github.com/SyntropyNet/syntropy-agent-go/config"
	"github.com/SyntropyNet/syntropy-agent-go/controller"
)

// Dummy controller just reads files from
// `/etc/syntropy-agent/dummy` directory and simulates communication with a controller
type DummmyController struct {
	list    []string
	timeout time.Duration
}

const dummyPath = config.AgentConfigDir + "/dummy"

// NewAgent allocates instance of agent struct
// Parses shell environment and setups internal variables
func NewController() (controller.Controller, error) {
	cc := DummmyController{
		timeout: 3 * time.Second,
	}

	script, err := ioutil.ReadFile(dummyPath + "/SCRIPT")
	if err != nil {
		files, err := ioutil.ReadDir(dummyPath)
		if err != nil {
			return nil, fmt.Errorf("could not initialise dummy controller: %s", err.Error())
		}
		for _, file := range files {
			cc.list = append(cc.list, file.Name())
		}
	} else {
		cc.list = strings.Split(string(script), "\n")
	}

	log.Println("Initialise dummy controller. Scripts: ", cc.list, len(cc.list), ".")
	return &cc, nil
}

// Start is main loop of SyntropyStack agent
func (cc *DummmyController) Start(rx, tx chan []byte) {

	// Receiver goroutine
	go func() {
		// Some delay before starting
		time.Sleep(cc.timeout)

		for _, fname := range cc.list {
			if fname == "" || fname[0] == '#' {
				continue
			}
			msg, err := ioutil.ReadFile(dummyPath + "/" + fname)
			if err != nil {
				log.Printf("DummyControler. File %s: %s", fname, err.Error())
				continue
			}
			rx <- []byte(msg)

			// delay before next message
			time.Sleep(cc.timeout)
		}
	}()

	// Sender goroutine
	go func() {
		for {
			// Just read channel. Discard and ignore sending messages
			_, ok := <-tx
			if !ok {
				return // terminate, because channel is closed
			}
		}
	}()

}

// Stop closes connection
func (cc *DummmyController) Stop() {
}
