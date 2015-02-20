package main

import (
	"flag"
	"fmt"
	"github.com/lizthegrey/apache-log-monitor/monitor"
)

var filename = flag.String("file", "/var/log/apache2/access.log", "The filename of the W3C formatted logfile to parse.")

func main() {
	flag.Parse()
	fileHandle := monitor.TailFile(*filename)
	lines := make(chan string)
	terminate := make(chan bool)
	go func() { fileHandle.ContinuousRead(lines) }()
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
