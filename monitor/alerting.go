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
	firstFired   *time.Time
}

func (a *ThresholdAlert) Evaluate(observed int64) *string {
	var firing bool
	var conditionString string
	if a.TriggerAbove {
		firing = observed >= a.Threshold
		conditionString = "High"
	} else {
		firing = observed <= a.Threshold
		conditionString = "Low"
	}

	now := time.Now()
	if a.firstFired == nil && firing {
		// We've newly started firing.
		a.firstFired = &now
		msg := fmt.Sprintf("%s traffic generated an alert - hits = %d, triggered at %s", conditionString, observed, a.firstFired.Format(time.UnixDate))
		return &msg
	}

	if a.firstFired != nil && !firing {
		// We've recovered.
		msg := fmt.Sprintf("%s traffic alert has recovered - hits = %d, last triggered at %s", conditionString, observed, a.firstFired.Format(time.UnixDate))
		a.firstFired = nil
		return &msg
	}

	return nil
}
