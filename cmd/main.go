package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/SyntropyNet/syntropy-agent-go/agent"
	"github.com/SyntropyNet/syntropy-agent-go/config"
)

const appName = "sag"

func main() {
	showVersionAndExit := flag.Bool("version", false, "Show version and exit")
	flag.Parse()
	if *showVersionAndExit {
		fmt.Printf("%s:\t%s\n\n", appName, config.GetFullVersion())
		return
	}

	log.Println(appName, config.GetFullVersion(), "started")

	config.Init()
	// TODO: init Wireguard (see pyroyte2.Wireguard())

	syntropyNetAgent, err := agent.NewAgent()
	if err != nil {
		log.Fatal("Could not create Syntropy Stack agent: ", err)
	}

	//Start main agent loop (forks to goroutines internally)
	syntropyNetAgent.Loop()

	// Wait for SIGINT or SIGKILL to terminate app
	terminate := make(chan os.Signal, 1)
	signal.Notify(terminate, os.Interrupt)
	<-terminate
	log.Println("SyntropyAgent terminating")

	// Stop and cleanup
	syntropyNetAgent.Stop()
}
