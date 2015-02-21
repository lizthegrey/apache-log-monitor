package main

import (
	"flag"
	"fmt"
	"github.com/lizthegrey/apache-log-monitor/monitor"
)

var filename = flag.String("file", "/var/log/apache2/access.log", "The filename of the W3C formatted logfile to parse.")
var pollIntervalMs = flag.Int64("poll_interval_ms", 250, "The number of milliseconds to wait between tail polling attempts.")

func main() {
	flag.Parse()
	lines := make(chan string)
	file, err := monitor.TailFile(*filename, lines, *pollIntervalMs)
	if err != nil {
		fmt.Println(err)
		return
	}
	terminate := make(chan bool)
	go func() {
		err := file.ContinuousRead()
		fmt.Println(err)
		// Signal cleanup if main read loop is over.
		file.Close()
	}()
	go func() {
		for {
			line, more := <-lines
			if !more {
				terminate <- true
				return
			}

			result, err := monitor.LogParse(line)
			if err == nil {
				fmt.Println(result.Url)
			} else {
				fmt.Println(err)
			}
		}
	}()
	// trigger console functionality here
	<-terminate
}
