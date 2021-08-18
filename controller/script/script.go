package script

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"strings"
	"time"

	"github.com/SyntropyNet/syntropy-agent-go/config"
	"github.com/SyntropyNet/syntropy-agent-go/controller"
)

// Script controller just reads files from
// `/etc/syntropy-agent/script` directory and setups accordinly
type ScriptController struct {
	list    []string
	index   int
	timeout time.Duration
	stop    chan bool
}

const scriptPath = config.AgentConfigDir + "/script"

// NewAgent allocates instance of agent struct
// Parses shell environment and setups internal variables
func NewController() (controller.Controller, error) {
	cc := ScriptController{
		timeout: 1 * time.Second,
	}

	script, err := ioutil.ReadFile(scriptPath + "/SCRIPT")
	if err != nil {
		files, err := ioutil.ReadDir(scriptPath)
		if err != nil {
			return nil, fmt.Errorf("could not initialise script controller: %s", err.Error())
		}
		for _, file := range files {
			cc.list = append(cc.list, file.Name())
		}
	} else {
		cc.list = strings.Split(string(script), "\n")
	}
	cc.stop = make(chan bool)

	log.Println("Initialise Script controller. Scripts: ", cc.list, len(cc.list), ".")
	return &cc, nil
}

func (cc *ScriptController) Recv() ([]byte, error) {
	// Some delay before starting
	time.Sleep(cc.timeout)
	for cc.index < len(cc.list) {
		fname := cc.list[cc.index]
		cc.index++
		if fname == "" || fname[0] == '#' {
			continue
		}
		msg, err := ioutil.ReadFile(scriptPath + "/" + fname)
		if err != nil {
			log.Printf("ScriptControler. File %s: %s", fname, err.Error())
			continue
		}

		return msg, nil
	}

	// When no more configuration scripts are left - just block the Recv
	// and keep agent waiting
	_, ok := <-cc.stop
	if !ok {
		return nil, io.EOF
	}
	return nil, fmt.Errorf("unexpected controller usage")
}

// Write sends nowhere
func (cc *ScriptController) Write(b []byte) (n int, err error) {
	return len(b), nil
}

// Close terminates connection
func (cc *ScriptController) Close() error {
	close(cc.stop)
	return nil
}