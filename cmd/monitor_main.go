package main

import (
	"flag"
	"github.com/lizthegrey/apache-log-monitor/monitor"
	"log"
	"os"
)

var filename = flag.String("file", "/var/log/apache2/access.log", "The filename of the W3C formatted logfile to parse.")
var pollIntervalMs = flag.Int64("poll_interval_ms", 250, "The number of milliseconds to wait between tail polling attempts.")

func main() {
	flag.Parse()
	log.SetOutput(os.Stderr)

	// Buffering limit of 5 picked out of hat. Definitely needs to be >1 but value should be experimentally determined.
	lines := make(chan string, 5)
	consoleStatus := make(chan string)
	consoleLog := make(chan string, 5)
	terminate := make(chan bool)

	file, err := monitor.TailFile(*filename, lines, *pollIntervalMs)
	if err != nil {
		log.Println(err)
		return
	}
	go func() {
		err := file.ContinuousRead()
		log.Println(err)
		// Signal cleanup if main read loop is over.
		file.Close()
	}()

	go func() {
		for line := range lines {
			result, err := monitor.LogParse(line)
			if err == nil {
				// TODO(lizf): Send the result object to all registered statistics modules,
				//             then add extra indirection layer for stats modules to write to console/status.
				consoleLog <- result.Url.String()
			} else {
				consoleLog <- err.Error()
			}
		}
		terminate <- true
	}()

	console, err := monitor.NewConsole(consoleStatus, consoleLog)
	if err != nil {
		log.Println(err)
		return
	}

	go func() {
		<-terminate
		console.Interrupt()
	}()

	defer console.Cleanup()
	if err = console.Loop(); err != nil {
		log.Println(err)
	}
	log.Println("Terminating...")
}
