package main

import (
	"flag"
	"github.com/lizthegrey/apache-log-monitor/monitor"
)

var filename = flag.String("file", "/var/log/apache2/access.log", "The filename of the W3C formatted logfile to parse.")

func main() {
	flag.Parse()
	monitor.LogParse(*filename)
}
