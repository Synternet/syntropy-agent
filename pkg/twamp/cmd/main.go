package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/SyntropyNet/syntropy-agent/pkg/twamp"
)

func main() {
	interval := flag.Int("interval", 1, "Delay between TWAMP-test requests (seconds)")
	count := flag.Int("count", 0, "Number of requests to send (1..2000000000 packets)")
	size := flag.Int("size", 42, "Size of request packets (0..65468 bytes)")
	tos := flag.Int("tos", 0, "IP type-of-service value (0..255)")
	wait := flag.Int("wait", 1, "Maximum wait time after sending final packet (seconds)")
	port := flag.Int("port", 6666, "UDP port to send request packets")

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

		client, err := twamp.NewClient(remoteIP,
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

		fmt.Printf("TWAMP PING %s: %d data bytes\n",
			client.GetHost(), 14+client.PaddingSize())

		i := 0
		t := time.NewTicker(time.Duration(*interval) * time.Second)
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)

		defer func() {
			t.Stop()
			Stats := client.GetStats()
			fmt.Printf("--- %s twamp ping statistics ---\n", client.GetHost())
			fmt.Printf("%d packets transmitted, %d packets received\n%0.1f%% packet loss %0.03f ms latency\n",
				Stats.Tx(), Stats.Rx(), Stats.Loss(), Stats.Latency())
			client.Close()
		}()

		for {
			select {
			case <-t.C:
				stats, err := client.Run()
				if err != nil {
					fmt.Println("error:", err)
				} else {
					fmt.Printf("recv from %s: twamp_seq=%d time=%0.03f ms\n",
						client.GetHost(), i, float32(stats.Rtt().Microseconds())/1000)
				}
				i++

				if *count > 0 && i >= *count {
					return
				}
			case <-c:
				return
			}
		}

	}

	if *server {
		runServer()
	} else {
		runClient()
	}
}
