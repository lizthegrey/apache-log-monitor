package main

import (
	"github.com/lizthegrey/apache-log-monitor/monitor"

	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

var filename = flag.String("file", "/var/log/apache2/access.log", "The filename of the W3C formatted logfile to parse.")
var pollIntervalMs = flag.Int64("poll_interval_ms", 250, "The number of milliseconds to wait between tail polling attempts.")

var bucketSizeS = flag.Int64("bucket_size_s", 1, "The number of seconds of granularity for data collection.")

var statsEvalS = flag.Int64("stats_eval_s", 1, "The number of seconds between computation of current statistics.")
var statsWindowS = flag.Int64("stats_window_s", 10, "The number of seconds for the sliding window for statistics computation.")

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
	trafficAlertStats := monitor.NewRing(int(*trafficWindowS / *bucketSizeS))
	consoleStats := monitor.NewRing(int(*statsWindowS / *bucketSizeS))
	pathStats := monitor.NewRing(int(*statsWindowS / *bucketSizeS))

	go func() {
		for {
			time.Sleep(time.Duration(*bucketSizeS) * time.Second)

			trafficAlertStats.Rotate()
			consoleStats.Rotate()
			pathStats.Rotate()
		}
	}()

	// Invoke the log parser.
	go func() {
		for line := range lines {
			result, err := monitor.LogParse(line)
			if err == nil {
				// Unconditionally increment the traffic stats mapping for each received query.
				trafficAlertStats.Mutate(func(s *monitor.Stat) { s.Map["all"] += 1 })

				// We want to record: QPS, bytes/sec, non-200s/sec, popular URL paths over status interval
				// (cue "what do you want to monitor about a webserver?")
				// We also want to track the scalar value of average processing delay between log entry arriving and us processing.
				consoleStats.Mutate(func(s *monitor.Stat) {
					s.Map["allReqs"] += 1
					s.Map["allBytes"] += result.Size
					if result.Status != 200 {
						s.Map["non200s"] += 1
					}
					s.Map["logLatencyMs"] += int64(time.Since(result.Timestamp) / time.Millisecond)
				})
				path := strings.SplitN(result.Url.Path, "/", 3)
				if len(path) >= 2 {
					pathStats.Mutate(func(s *monitor.Stat) { s.Map[path[1]] += 1 })
				}
			} else {
				consoleLog <- err.Error()
			}
		}
		terminate <- true
	}()

	// Invoke the console stats loop.
	go func() {
		for {
			consoleStats.Mtx.Lock()
			count := float64(consoleStats.Sum.Map["allReqs"])
			bytes := float64(consoleStats.Sum.Map["allBytes"])
			non200s := float64(consoleStats.Sum.Map["non200s"])
			latencyAvgMs := float64(consoleStats.Sum.Map["logLatencyMs"]) / count
			consoleStats.Mtx.Unlock()

			pathStats.Mtx.Lock()
			var highestVal int64 = 0
			var highestPath string = ""
			for path, hits := range pathStats.Sum.Map {
				if hits > highestVal {
					highestVal = hits
					highestPath = fmt.Sprintf("/%s", path)
				}
			}
			pathStats.Mtx.Unlock()

			// Construct line and send to status display.
			window := float64(*statsWindowS)
			basic := fmt.Sprintf("QPS %.1f, Bytes/sec %.1f, Errs/sec %.1f, ErrRatio %.2f, ", count/window, bytes/window, non200s/window, non200s/count)
			var popular string
			if highestVal > 0 {
				popular = fmt.Sprintf("Popular %s (%d queries over %.fs window), ", highestPath, highestVal, window)
			} else {
				popular = fmt.Sprintf("Popular - (no traffic over %.fs window), ", window)
			}
			delay := fmt.Sprintf("Delay(ms) %.f", latencyAvgMs)

			line := fmt.Sprint(basic, popular, delay)
			consoleStatus <- line

			time.Sleep(time.Duration(*statsEvalS) * time.Second)
		}
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
			if msg := high.Evaluate(trafficAlertStats.SumLookup("all")); msg != nil {
				consoleLog <- *msg
			}
			if msg := low.Evaluate(trafficAlertStats.SumLookup("all")); msg != nil {
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
