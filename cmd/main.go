package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"

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

	config.Init()
	defer config.Close()

	syntropyNetAgent, err := agent.NewAgent(config.GetControllerType())
	if err != nil {
		log.Fatal("Could not create ", fullAppName, err)
	}

	logger.Info().Println(fullAppName, execName, config.GetFullVersion(), "started.")
	logger.Info().Println(fullAppName, "Using controller type: ", config.GetControllerType())

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
