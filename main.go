package main

import (
	"log"
	"os"
	"os/signal"

	"github.com/SyntropyNet/syntropy-agent-go/agent"
)

const appName = "sag"
const version = "0.0.69"

func main() {
	log.Println(appName, version, "started")

	// TODO: init Wireguard (see pyroyte2.Wireguard())

	agent, err := agent.NewAgent(version)
	if err != nil {
		log.Fatal("Could not create Syntropy Stack agent: ", err)
	}

	log.Println("Connected")

	go agent.Loop()

	terminate := make(chan os.Signal, 1)
	signal.Notify(terminate, os.Interrupt)
	<-terminate
	log.Println("SyntropyAgent terminating")
	agent.Stop()
}
