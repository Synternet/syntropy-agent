package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"os/user"
	"time"

	"github.com/SyntropyNet/syntropy-agent-go/agent"
	"github.com/SyntropyNet/syntropy-agent-go/internal/config"
	"github.com/SyntropyNet/syntropy-agent-go/internal/env"
	"github.com/SyntropyNet/syntropy-agent-go/internal/logger"
	"golang.org/x/sys/unix"
)

const (
	fullAppName = "Syntropy Stack Agent. "
	lockFile    = "/var/lock/syntropy"
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

	// Perform locking using Flock.
	// If running from docker - it is recommended to use `-v /var/lock/syntropy:/var/lock/syntropy`
	f, err := os.Create(lockFile)
	if err != nil {
		logger.Error().Println(fullAppName, lockFile, err)
		exitCode = -2 // errno.h ENOENT
		return
	}
	err = unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB)
	if err != nil {
		// Another agent instance is running. Exit.
		logger.Error().Println(fullAppName, "Another agent instance is running")
		logger.Error().Println(fullAppName, "Lock file residual", lockFile)
		exitCode = -16 // errno.h -EBUSY
		return
	}
	defer func() {
		unix.Flock(int(f.Fd()), unix.LOCK_UN)
		f.Close()
		os.Remove(lockFile)
	}()

	config.Init()
	defer config.Close()

	syntropyNetAgent, err := agent.New(config.GetControllerType())
	if err != nil {
		logger.Error().Println(fullAppName, "Could not create agent", err)
		exitCode = -12 // errno.h -ENOMEM
		return
	}

	logger.Info().Println(fullAppName, execName, config.GetFullVersion(), "started.")
	logger.Info().Println(fullAppName, "Using controller type: ", config.GetControllerName(config.GetControllerType()))
	logger.Info().Println(fullAppName, "Local time:", time.Now().Format(env.TimeFormat))

	//Start main agent loop
	go syntropyNetAgent.Run()
	defer syntropyNetAgent.Close()

	// Wait for SIGINT or SIGKILL to terminate app
	terminate := make(chan os.Signal, 1)
	signal.Notify(terminate, os.Interrupt)
	<-terminate
	logger.Info().Println(fullAppName, " terminating")
}
