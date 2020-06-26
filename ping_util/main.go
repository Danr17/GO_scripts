package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	cliping "github.com/Danr17/GO_scripts/tree/master/ping_util/pkg/cli_ping"
	"github.com/Danr17/GO_scripts/tree/master/ping_util/pkg/utils"
	webping "github.com/Danr17/GO_scripts/tree/master/ping_util/pkg/web_ping"
)

const usage = `WEB:
1. Web Ping with defauts (defaults: port=443, interval=1s, counter=4)
    > tcp_ping-linux-amd64 -web -file=example.txt
2. Web Ping with over port=80, interval=3s, counter=10
    > tcp_ping-linux-amd64 -web -file=example.txt -port 80 -interval 3s -counter 10

CLI:
1. ping over tcp  with defaults (defaults: port=443, interval=1s, counter=4)
    > tcp_ping-linux-amd64 example.com
2. ping over tcp over custom port
    > tcp_ping-linux-amd64 -port 22 example.com 
3. ping over tcp using counter and interval
    > tcp_ping-linux-amd64 -port 80 -counter 3 -interval 3s example.com
`

var (
	proto     = flag.String("p", "tcp", "enable this if you want to see it on Web")
	port      = flag.Int("port", 443, "enable this if you want to see it on Web")
	inWebFile = flag.String("file", "", "specify the filename")
	counter   = flag.Int("c", 4, "ping counter")
	timeout   = flag.String("timeout", "1s", `connect timeout, units are "ns", "us" (or "µs"), "ms", "s", "m", "h"`)
	interval  = flag.String("interval", "1s", `ping interval, units are "ns", "us" (or "µs"), "ms", "s", "m", "h"`)
)

//Protocol not used yet
type Protocol int

const (
	tcp Protocol = iota
	udp
	icmp
	web
)

func main() {
	flag.Parse()
	args := flag.Args()

	//catch CTRL-C
	sigs := make(chan os.Signal, 1)
	//notify web to stop
	done := make(chan bool, 1)
	//channel to listen for errors coming from the listener.
	serverErrors := make(chan error, 1)

	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	timeoutDuration, err := utils.ConvertTime(*timeout)
	if err != nil {
		log.Fatalln("The value provided for **timeout** is wrong")
	}
	intervalDuration, err := utils.ConvertTime(*interval)
	if err != nil {
		log.Fatalln("The value provided for **interval** is wrong")
	}

	var server *http.Server
	var WEBpinger *webping.WebPing
	var CLIpinger *cliping.CLIping

	switch *proto {
	case "tcp", "udp":
		CLIpinger = cliping.NewCLIping()
		done = startCLI(CLIpinger, args, timeoutDuration, intervalDuration)
	case "icmp":
		done = cliping.StartICMP(args, *counter, intervalDuration, timeoutDuration)

	case "web":
		if *inWebFile == "" {
			fmt.Println(usage)
		}

		server, WEBpinger = startWeb(args, timeoutDuration, intervalDuration)

		go func() {
			log.Println("API listening on port 8080. Open browser: http://localhost:8080")
			serverErrors <- server.ListenAndServe()
		}()

		go WEBpinger.Start(done)
	default:
		log.Panicln("The value provided for protocol is not valid, should be --p web, --p tcp, --p udp or --p icmp")

	}

	select {
	case err := <-serverErrors:
		log.Fatalf("error: starting server: %s", err)
	case <-sigs:
		if *proto == "web" {
			log.Println("Closing HTTP server")
			server.Close()
			close(done)
			return
		}
		CLIpinger.Stop()
		return
	case <-done:
		return
	}

}
