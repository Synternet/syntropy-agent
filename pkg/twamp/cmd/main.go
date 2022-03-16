package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/SyntropyNet/syntropy-agent/pkg/twamp"
)

func main() {
	interval := flag.Int("interval", 1, "Delay between TWAMP-test requests (seconds)")
	count := flag.Int("count", 5, "Number of requests to send (1..2000000000 packets)")
	rapid := flag.Bool("rapid", false, "Send requests rapidly (default count of 5)")
	size := flag.Int("size", 42, "Size of request packets (0..65468 bytes)")
	tos := flag.Int("tos", 0, "IP type-of-service value (0..255)")
	wait := flag.Int("wait", 1, "Maximum wait time after sending final packet (seconds)")
	port := flag.Int("port", 6666, "UDP port to send request packets")
	mode := flag.String("mode", "ping", "Mode of operation (ping, json)")

	server := flag.Bool("server", false, "Start a TWAMP server (default is client mode)")
	listenPtr := flag.String("listen", "localhost", "listen address")
	udpStart := flag.Uint("udp-start", 2000, "initial UDP port for tests")

	flag.Parse()

	runServer := func() {
		s, err := twamp.NewServer(*listenPtr, uint16(*udpStart))
		if err != nil {
			fmt.Println("Error starting server", err)
			return
		}

		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)

		go func() {
			sig := <-c
			fmt.Println("Exiting, got signal:", sig)
			s.Close()
		}()

		err = s.Run()
		if err != nil {
			log.Println(err)
		}
	}

	runClient := func() {
		args := flag.Args()

		if len(args) < 1 {
			fmt.Println("No hostname or IP address was specified.")
			os.Exit(1)
		}

		remoteIP := args[0]
		remoteServer := fmt.Sprintf("%s:%d", remoteIP, twamp.TwampControlPort)

		c := twamp.NewClient()
		connection, err := c.Connect(remoteServer)
		if err != nil {
			log.Fatal(err)
		}

		session, err := connection.CreateSession(
			twamp.SessionConfig{
				Port:    *port,
				Timeout: *wait,
				Padding: *size,
				TOS:     *tos,
			},
		)
		if err != nil {
			log.Fatal(err)
		}

		test, err := session.CreateTest()
		if err != nil {
			log.Fatal(err)
		}

		switch *mode {
		case "json":
			results := test.RunX(*count)
			test.FormatJSON(results)
		case "ping":
			test.Ping(*count, *rapid, *interval)
		}

		session.Stop()
		connection.Close()
	}

	if *server {
		runServer()
	} else {
		runClient()
	}
}
