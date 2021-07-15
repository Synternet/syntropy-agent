package main

import (
	"log"

	"github.com/SyntropyNet/syntropy-agent-go/app/agent"
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
	defer agent.Close()

	agent.Run()

}
