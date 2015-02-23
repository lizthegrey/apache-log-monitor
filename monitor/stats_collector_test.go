package monitor

import "testing"

func TestRingBufferStats(t *testing.T) {
	cases := []struct {
		rotateBefore bool
		increment    int64
		sum          int64
	}{
		{false, 0, 0}, // {0, {0, 0, 0}}
		{false, 1, 1}, // {1, {1, 0, 0}}
		{false, 2, 3}, // {3, {3, 0, 0}}
		{true, 1, 4},  // {4, {1, 3, 0}}
		{false, 5, 9}, // {9, {6, 3, 0}}
		{true, 2, 11}, // {11, {2, 6, 3}}
		{true, 0, 8},  // {8, {0, 2, 6}}
		{false, 1, 9}, // {9, {1, 2, 6}}
		{true, 0, 3},  // {7, {0, 1, 2}}
		{true, 0, 1},  // {1, {0, 0, 1}}
		{true, 0, 0},  // {0, {0, 0, 0}}
	}

	r := NewRing(3)

	// Repeatedly add to the ring, making sure rotation works as intended.
	for _, c := range cases {
		if c.rotateBefore {
			r.Rotate()
		}
		r.Mutate(func(v *Stat) {
			if sum := v.Map["test"] + c.increment; sum != 0 {
				v.Map["test"] = sum
			}
		})
		if total := r.Sum.Map["test"]; c.sum != total {
			t.Errorf("Sum - Expected: %d, Got: %d", c.sum, total)
		}
	}
	if _, found := r.Sum.Map["test"]; found {
		t.Errorf("Leaking memory - 0 values should be deleted.")
	}
}
