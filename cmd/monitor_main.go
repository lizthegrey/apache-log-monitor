package main

import (
	"flag"
	"github.com/lizthegrey/apache-log-monitor/monitor"
	"log"
	"os"
	"time"
)

var filename = flag.String("file", "/var/log/apache2/access.log", "The filename of the W3C formatted logfile to parse.")
var pollIntervalMs = flag.Int64("poll_interval_ms", 250, "The number of milliseconds to wait between tail polling attempts.")

var bucketSizeS = flag.Int64("bucket_size_s", 1, "The number of seconds of granularity for data collection.")
var statsEvalS = flag.Int64("stats_eval_s", 10, "The number of seconds between computation of current statistics.")
var alertEvalS = flag.Int64("alert_eval_s", 5, "The number of seconds between evaluation of alert conditions.")
var highTrafficThresholdQps = flag.Int64("high_traffic_threshold_qps", 1, "The QPS value above which we will generate an alert.")
var lowTrafficThresholdQps = flag.Int64("low_traffic_threshold_qps", 0, "The QPS value below which we will generate an alert.")
var trafficWindowS = flag.Int64("traffic_window_s", 15, "The number of seconds for the sliding window for QPS alerts.")

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

	// Define our statistics objects, and set them to rotate on a granular basis.
	// Traffic alerting needs to expire buckets every bucketSizeS seconds,
	// and keep a total number of buckets corresponding to the window.
	trafficStats := monitor.NewRing(int(*trafficWindowS / *bucketSizeS))
	go func() {
		for {
			time.Sleep(time.Duration(*bucketSizeS) * time.Second)
			trafficStats.Rotate()
		}
	}()

	// Invoke the log parser.
	go func() {
		for line := range lines {
			// TODO(lizf): put back result once I have a use for it.
			_, err := monitor.LogParse(line)
			if err == nil {
				// TODO(lizf): Send the result object to all registered statistics modules,
				//             then add extra indirection layer for stats modules to write to console/status.

				// Unconditionally increment the traffic stats mapping.
				trafficStats.Mutate(func(s *monitor.Stat) { s.Map["all"] += 1 })

				// We want to record: QPS, bytes/sec, non-200s/sec, popular URL paths over status interval
				// (cue "what do you want to monitor about a webserver?")
				// We also want to track the scalar value of average processing delay between log entry arriving and us processing.
			} else {
				consoleLog <- err.Error()
			}
		}
		terminate <- true
	}()

	// Invoke the alert evaluation loop.
	go func() {
		high := monitor.ThresholdAlert{
			Threshold:    *highTrafficThresholdQps * *trafficWindowS,
			TriggerAbove: true,
		}
		low := monitor.ThresholdAlert{
			Threshold:    *lowTrafficThresholdQps * *trafficWindowS,
			TriggerAbove: false,
		}
		for {
			time.Sleep(time.Duration(*alertEvalS) * time.Second)
			if msg := high.Evaluate(trafficStats.SumLookup("all")); msg != nil {
				consoleLog <- *msg
			}
			if msg := low.Evaluate(trafficStats.SumLookup("all")); msg != nil {
				consoleLog <- *msg
			}
		}
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
