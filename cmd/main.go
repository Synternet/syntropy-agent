package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"os/user"
	"strconv"
	"strings"

	"github.com/SyntropyNet/syntropy-agent-go/agent"
	"github.com/SyntropyNet/syntropy-agent-go/internal/config"
	"github.com/SyntropyNet/syntropy-agent-go/internal/logger"
)

const (
	fullAppName = "Syntropy Stack Agent. "
	lockFile    = "/var/lock/syntropy_agent.lock"
)

func requireRoot() {
	user, err := user.Current()
	if err != nil {
		logger.Error().Println(fullAppName, "current user", err)
		os.Exit(-14) // errno.h -EFAULT
	} else if user.Uid != "0" {
		logger.Error().Println(fullAppName, "insufficient permitions. Please run with `sudo` or as root.")
		os.Exit(-13) // errno.h -EACCES
	}
}

func agentLock() {
	pidStr, _ := ioutil.ReadFile(lockFile)
	pid, _ := strconv.Atoi(strings.ReplaceAll(string(pidStr), "\n", ""))

	if pid > 0 {
		_, err := os.Stat(fmt.Sprintf("/proc/%d", pid))
		if err == nil {
			// Another agent instance is running. Exit.
			logger.Error().Println(fullAppName, "Another agent instance is running")
			logger.Error().Println(fullAppName, "check lock file", lockFile)
			os.Exit(-16) // errno.h -EBUSY
		} else {
			// Agent is not running. Just for some reasons lock file is present. Continue.
			logger.Warning().Println(fullAppName, "residual lock file found. An agent was killed or crashed before?")
		}
	}

	ioutil.WriteFile(lockFile, []byte(strconv.Itoa(os.Getpid())), 0644)
}

func agentUnlock() {
	os.Remove(lockFile)
}

func main() {
	exitCode := 0
	defer func() { os.Exit(exitCode) }()

	execName := os.Args[0]

	showVersionAndExit := flag.Bool("version", false, "Show version and exit")

	flag.Parse()
	if *showVersionAndExit {
		fmt.Printf("%s (%s):\t%s\n\n", fullAppName, execName, config.GetFullVersion())
		return
	}

	requireRoot()
	agentLock()
	defer agentUnlock()

	config.Init()
	defer config.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	syntropyNetAgent, err := agent.NewAgent(ctx, config.GetControllerType())
	if err != nil {
		logger.Error().Println(fullAppName, "Could not create agent", err)
		exitCode = -12 // errno.h -ENOMEM
		return
	}

	logger.Info().Println(fullAppName, execName, config.GetFullVersion(), "started.")
	logger.Info().Println(fullAppName, "Using controller type: ", config.GetControllerName(config.GetControllerType()))

	//Start main agent loop
	go func() {
		if err := syntropyNetAgent.Run(); err != nil {
			cancel()
		}
	}()

	// Wait for SIGINT or SIGKILL to terminate app
	terminate := make(chan os.Signal, 1)
	signal.Notify(terminate, os.Interrupt)
	<-terminate
	logger.Info().Println(fullAppName, " terminating")
}
