package main

import (
	"errors"
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
	"github.com/beevik/ntp"
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

// checkTime gets current time from NTP servers pool
// and checks if local time is accurate.
// Prints warning to log if time differs by more than 10 seconds
func checkTime() {
	const toleratedDiff = 10 * time.Second

	// Default options have 5 second timeout before error
	ntpTime, err := ntp.Time("pool.ntp.org")
	if err != nil {
		logger.Error().Println(fullAppName, "NTP error", err)
		return
	}

	localTime := time.Now()
	timeDiff := localTime.Sub(ntpTime)

	if timeDiff > toleratedDiff || timeDiff < -toleratedDiff {
		logger.Warning().Println(fullAppName, "Inaccurate time. Local:", localTime.Format(time.Stamp),
			"NTP:", ntpTime.Format(time.Stamp))
	}
}

func requireProcFilesystemWritable() {
	f, err := os.OpenFile("/proc/sys/net/ipv4/ip_forward", os.O_WRONLY|os.O_TRUNC, 0611)
	if errors.Is(err, unix.EROFS) {
		logger.Error().Println(fullAppName, "/proc/sys filesystem is readonly. If on docker, restart with '--privileged' flag.")
		os.Exit(-int(unix.EROFS))
	} else if err != nil {
		logger.Error().Println(fullAppName, "cannot access /proc/sys filesystem", err)
		os.Exit(-int(unix.EACCES))
	}
	f.Close()
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
	requireProcFilesystemWritable()

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

	logger.SetupGlobalLoger(nil, config.GetDebugLevel(), os.Stdout)

	syntropyNetAgent, err := agent.New(config.GetControllerType())
	if err != nil {
		logger.Error().Println(fullAppName, "Could not create agent", err)
		checkKernelVersion()
		exitCode = -int(unix.ENOMEM)
		return
	}

	if config.GetControllerType() == config.ControllerSaas {
		// Config loggers early - to get more info logged
		// NOTE: Setup remote logger only for saas controller
		logger.SetupGlobalLoger(syntropyNetAgent.Writer(), config.GetDebugLevel(), os.Stdout)
	}

	logger.Exec().Println(fullAppName, execName, config.GetFullVersion(), "started.")
	logger.Info().Println(fullAppName, "Using controller type: ", config.GetControllerName(config.GetControllerType()))
	checkKernelVersion()
	checkTime()

	//Start main agent loop
	go syntropyNetAgent.Run()
	defer syntropyNetAgent.Close()

	// Wait for SIGINT or SIGKILL to terminate app
	terminate := make(chan os.Signal, 1)
	signal.Notify(terminate, syscall.SIGINT, syscall.SIGTERM)
	<-terminate
	logger.Exec().Println(fullAppName, " terminating")
}
