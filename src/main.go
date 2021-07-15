package main

import (
	"log"
	"github.com/SyntropyNet/syntropy-agent-go/app/agent"
)

const appName = "sag"
const version = "v0.0.1"

func main() {
	log.Println(appName, version, "started")

	// TODO: init Wireguard (see pyroyte2.Wireguard())


	agent, err := agent.NewAgent()
	if err != nil {
		log.Fatalf("Could not create Syntropy Stack agent", err)
	}

	agent.Run()

}
