package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"os/user"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/SyntropyNet/syntropy-agent/agent"
	"github.com/SyntropyNet/syntropy-agent/internal/config"
	"github.com/SyntropyNet/syntropy-agent/internal/env"
	"github.com/SyntropyNet/syntropy-agent/internal/logger"
	"golang.org/x/sys/unix"
)

const (
	fullAppName = "Syntropy Stack Agent. "
)

func requireRoot() {
	user, err := user.Current()
	if err != nil {
		logger.Error().Println(fullAppName, "current user", err)
		os.Exit(-int(unix.EFAULT))
	} else if user.Uid != "0" {
		logger.Error().Println(fullAppName, "insufficient permitions. Please run with `sudo` or as root.")
		os.Exit(-int(unix.EACCES))
	}
}

func checkKernelVersion() {
	// Linux kernel 5.6 was first to have wireguard included in kernel
	// If version is lower - log warning and try to continue
	const minVer = 5
	const minSubver = 6

	utsname := new(unix.Utsname)
	unix.Uname(utsname)
	str := string(utsname.Release[0:])
	parts := strings.Split(str, ".")
	if len(parts) < 2 {
		logger.Warning().Println(fullAppName, "Strange kernel version", str)
		return
	}
	v1, _ := strconv.Atoi(parts[0])
	v2, _ := strconv.Atoi(parts[1])

	if v1 < minVer || (v1 == minVer && v2 < minSubver) {
		logger.Warning().Println(fullAppName, "Kernel version is:", str)
		logger.Warning().Println(fullAppName, "Some features may be not fully supported.")
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
		checkKernelVersion()
		return
	}

	requireRoot()

	// Perform locking using Flock.
	// If running from docker - it is recommended to use `-v /var/lock/syntropy:/var/lock/syntropy`
	f, err := os.Create(env.LockFile)
	if err != nil {
		logger.Error().Println(fullAppName, env.LockFile, err)
		exitCode = -int(unix.ENOENT)
		return
	}
	err = unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB)
	if err != nil {
		// Another agent instance is running. Exit.
		logger.Error().Println(fullAppName, "Another agent instance is running")
		logger.Error().Println(fullAppName, "Lock file residual", env.LockFile)
		exitCode = -int(unix.EBUSY)
		return
	}
	defer func() {
		unix.Flock(int(f.Fd()), unix.LOCK_UN)
		f.Close()
		os.Remove(env.LockFile)
	}()

	// Create required directories
	env.Init()
	// Parse configuration environment variables
	config.Init()
	defer config.Close()

	syntropyNetAgent, err := agent.New(config.GetControllerType())
	if err != nil {
		logger.Error().Println(fullAppName, "Could not create agent", err)
		checkKernelVersion()
		exitCode = -int(unix.ENOMEM)
		return
	}

	logger.Info().Println(fullAppName, execName, config.GetFullVersion(), "started.")
	logger.Info().Println(fullAppName, "Using controller type: ", config.GetControllerName(config.GetControllerType()))
	logger.Info().Println(fullAppName, "Local time:", time.Now().Format(env.TimeFormat))
	checkKernelVersion()

	//Start main agent loop
	go syntropyNetAgent.Run()
	defer syntropyNetAgent.Close()

	// Wait for SIGINT or SIGKILL to terminate app
	terminate := make(chan os.Signal, 1)
	signal.Notify(terminate, syscall.SIGINT, syscall.SIGTERM)
	<-terminate
	logger.Info().Println(fullAppName, " terminating")
}
