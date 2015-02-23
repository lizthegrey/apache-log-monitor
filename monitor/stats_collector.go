package monitor

import (
	"container/ring"
	"log"
	"sync"
)

// Our goal is to retain data only for the maximum lifetime required to
// generate statistics for that bucketing scheme.
//
// We are maintain cumulative statistics for the entire ring buffer as well
// as statistics for the current bucket. That way, when this bucket reaches
// the end of the buffer and needs to be expired, we can subtract its stats
// from the cumulative stats and insert a new empty bucket into the ring.

type Stat struct {
	Map map[string]int64
}

func NewEmptyStat() *Stat {
	val := make(map[string]int64)
	return &Stat{val}
}

func (minuend *Stat) Subtract(subtrahend *Stat) {
	for key, sValue := range subtrahend.Map {
		if mValue, ok := minuend.Map[key]; ok {
			if difference := mValue - sValue; difference != 0 {
				minuend.Map[key] = difference
			} else {
				// Prune 0 entries from map.
				delete(minuend.Map, key)
			}
		} else {
			log.Panicf("Could not find key %s in minuend map.", key)
		}
	}
}

// Note: any attempts to read running-sum statistics must take Mtx first.
type RingBufferStats struct {
	Sum     *Stat
	buckets *ring.Ring
	Mtx     sync.Mutex
}

// Given a size and a zeroed instance of a stats object, populate the ring.
func NewRing(size int) RingBufferStats {
	r := ring.New(size)
	// Initialize the ring with all zero elements.
	for i := 0; i < size; i++ {
		// Make a pointer to a new stats bucket.
		r.Value = NewEmptyStat()
		r = r.Next()
	}
	return RingBufferStats{
		Sum:     NewEmptyStat(),
		buckets: r,
	}
}

func (r *RingBufferStats) Rotate() {
	r.Mtx.Lock()
	defer r.Mtx.Unlock()

	// Remove the expiring bucket's stats from the running totals.
	r.Sum.Subtract(r.buckets.Next().Value.(*Stat))

	// Move the current pointer up, and empty the bucket.
	r.buckets = r.buckets.Next()
	r.buckets.Value = NewEmptyStat()
}

// Takes required locks, then applies the mutation operation to running total
// and to the current bucket.
func (r *RingBufferStats) Mutate(f func(*Stat)) {
	r.Mtx.Lock()
	defer r.Mtx.Unlock()

	f(r.Sum)
	f(r.buckets.Value.(*Stat))
}
