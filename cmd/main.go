package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"os/user"

	"github.com/SyntropyNet/syntropy-agent-go/agent"
	"github.com/SyntropyNet/syntropy-agent-go/internal/config"
	"github.com/SyntropyNet/syntropy-agent-go/internal/logger"
)

const fullAppName = "Syntropy Stack Agent. "

func main() {
	execName := os.Args[0]

	showVersionAndExit := flag.Bool("version", false, "Show version and exit")
	flag.Parse()
	if *showVersionAndExit {
		fmt.Printf("%s (%s):\t%s\n\n", fullAppName, execName, config.GetFullVersion())
		return
	}

	user, err := user.Current()
	if err != nil {
		logger.Error().Println(fullAppName, "current user", err)
		os.Exit(-14) // errno.h -EFAULT
	} else if user.Uid != "0" {
		logger.Error().Println(fullAppName, "insufficient permitions. Please run with `sudo` or as root.")
		os.Exit(-13) // errno.h -EACCES
	}

	config.Init()
	defer config.Close()

	syntropyNetAgent, err := agent.NewAgent(config.GetControllerType())
	if err != nil {
		logger.Error().Println(fullAppName, "Could not create agent", err)
		os.Exit(-12) // errno.h -ENOMEM
	}

	logger.Info().Println(fullAppName, execName, config.GetFullVersion(), "started.")
	logger.Info().Println(fullAppName, "Using controller type: ", config.GetControllerName(config.GetControllerType()))

	//Start main agent loop (forks to goroutines internally)
	syntropyNetAgent.Loop()

	// Wait for SIGINT or SIGKILL to terminate app
	terminate := make(chan os.Signal, 1)
	signal.Notify(terminate, os.Interrupt)
	<-terminate
	logger.Info().Println(fullAppName, " terminating")

	// Stop and cleanup
	syntropyNetAgent.Stop()
}
