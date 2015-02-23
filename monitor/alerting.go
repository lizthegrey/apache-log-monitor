package monitor

import (
	"fmt"
	"time"
)

// TrafficAlert encapsulates all data required to do alert evaluation.
// Alerts with the same window can share the same stats object.
// Note that scheduling periodic evaluation should be done in a goroutine
// by a caller rather than in here.
// TODO(lizf): wait to alert if we have insufficient data for entire window.
type ThresholdAlert struct {
	Threshold    int64
	TriggerAbove bool // False if we should trigger if we drop below.
	lastFired    *time.Time
	Stats        *RingBufferStats
	StatsKey     string
	Output       chan string
}

func (a *ThresholdAlert) Evaluate() {
	a.Stats.Mtx.Lock()
	total := a.Stats.Sum.Map[a.StatsKey]
	a.Stats.Mtx.Unlock()

	var firing bool
	var conditionString string
	if a.TriggerAbove {
		firing = total >= a.Threshold
		conditionString = "High"
	} else {
		firing = total <= a.Threshold
		conditionString = "Low"
	}

	now := time.Now()
	if a.lastFired == nil && firing {
		// We've newly started firing.
		a.lastFired = &now
		msg := fmt.Sprintf("%s traffic generated an alert - hits = %d, triggered at %s", conditionString, total, a.lastFired.Format(time.UnixDate))
		a.Output <- msg
		return
	}

	if a.lastFired != nil && !firing {
		// We've recovered.
		msg := fmt.Sprintf("%s traffic alert has recovered - hits = %d, last triggered at %s", conditionString, total, a.lastFired.Format(time.UnixDate))
		a.Output <- msg
		a.lastFired = nil
		return
	}
}
