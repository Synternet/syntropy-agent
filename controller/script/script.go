package script

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"strconv"
	"strings"
	"time"

	"github.com/SyntropyNet/syntropy-agent/controller"
	"github.com/SyntropyNet/syntropy-agent/internal/env"
	"github.com/SyntropyNet/syntropy-agent/internal/logger"
)

const pkgName = "ScriptController. "

// Script controller just reads files from
// `/etc/syntropy-agent/script` directory and setups accordinly
type ScriptController struct {
	list    []string
	index   int
	timeout time.Duration
	ctx     context.Context
	cancel  context.CancelFunc
}

const scriptPath = env.AgentConfigDir + "/script"

// NewAgent allocates instance of agent struct
// Parses shell environment and setups internal variables
func New() (controller.Controller, error) {
	cc := ScriptController{
		timeout: 1 * time.Second,
	}
	cc.ctx, cc.cancel = context.WithCancel(context.Background())

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

	return &cc, nil
}

func delay(cmd string) {
	arr := strings.Fields(cmd)
	seconds, err := strconv.Atoi(arr[1])
	if err != nil {
		logger.Error().Println(pkgName, "Sleep errror", err)
		return
	}
	delay := time.Duration(seconds) * time.Second
	logger.Debug().Println(pkgName, "Sleep", delay)
	time.Sleep(delay)
}

func (cc *ScriptController) Recv() ([]byte, error) {
	// Some delay before starting
	time.Sleep(cc.timeout)
	for cc.index < len(cc.list) {
		fname := cc.list[cc.index]
		cc.index++
		if fname == "" || fname[0] == '#' {
			logger.Debug().Printf("%s Skip \"%s\"\n", pkgName, fname)
			continue
		}
		if strings.HasPrefix(fname, "sleep") {
			delay(fname)
			continue
		}
		msg, err := ioutil.ReadFile(scriptPath + "/" + fname)
		if err != nil {
			logger.Error().Printf("%s File %s: %s", pkgName, fname, err.Error())
			continue
		}
		logger.Debug().Printf("%s Receiving \"%s\"\n", pkgName, fname)

		return msg, nil
	}

	// When no more configuration scripts are left - just block the Recv
	// and keep agent waiting
	logger.Debug().Println(pkgName, "No more messages.")
	<-cc.ctx.Done()
	logger.Debug().Println(pkgName, "EOF")
	return nil, io.EOF
}

// Write sends nowhere
func (cc *ScriptController) Write(b []byte) (n int, err error) {
	return len(b), nil
}

// Close terminates connection
func (cc *ScriptController) Close() error {
	logger.Info().Println(pkgName, "Closing.")
	cc.cancel()
	return nil
}
