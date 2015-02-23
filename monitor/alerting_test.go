package monitor

import (
	"testing"

	"strings"
	"time"
)

func TestAlertOutput(t *testing.T) {
	cases := []struct {
		// Arguments
		threshold        int64
		triggerAbove     bool
		observed         int64
		previouslyFiring bool

		// Postconditions
		msgPrefix         string
		firstFiredCleared bool
		firstFiredSet     bool
	}{
		{10, true, 9, false, "", false, false},
		{10, true, 10, false, "High traffic generated an alert", false, true},
		{10, true, 11, false, "High traffic generated an alert", false, true},
		{10, false, 9, false, "Low traffic generated an alert", false, true},
		{10, false, 10, false, "Low traffic generated an alert", false, true},
		{10, false, 11, false, "", false, false},
		{10, true, 9, true, "High traffic alert has recovered", true, false},
		{10, true, 10, true, "", false, false},
		{10, true, 11, true, "", false, false},
		{10, false, 9, true, "", false, false},
		{10, false, 10, true, "", false, false},
		{10, false, 11, true, "Low traffic alert has recovered", true, false},
	}

	for _, c := range cases {
		a := ThresholdAlert{
			Threshold:    c.threshold,
			TriggerAbove: c.triggerAbove,
		}
		now := time.Now()
		if c.previouslyFiring {
			a.firstFired = &now
		}
		msg := a.Evaluate(c.observed)

		if c.firstFiredCleared && a.firstFired != nil {
			t.Errorf("Should have cleared firing bit for testcase %s", c)
		}
		if c.firstFiredSet && (a.firstFired == nil || a.firstFired == &now) {
			t.Errorf("Should have set new firing bit for testcase %s", c)
		}

		if msg == nil {
			if c.msgPrefix != "" {
				t.Errorf("Expected an alert with prefix '%s' and instead saw no alert.", c.msgPrefix)
			}
			continue
		}
		if !strings.HasPrefix(*msg, c.msgPrefix) {
			t.Errorf("Expected an alert with prefix '%s' and instead saw '%s'.", c.msgPrefix, *msg)
		}
	}
}
