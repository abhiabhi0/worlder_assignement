package generator

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"sync/atomic"
	"time"
)

type Reading struct {
	Value      float64   `json:"value"`
	SensorType string    `json:"sensor_type"`
	ID1        string    `json:"id1"`
	ID2        int       `json:"id2"`
	Timestamp  time.Time `json:"timestamp"`
}

// Loop generates readings at a configurable frequency and prints each reading
// as a single JSON line to stdout (the "stream").
type Loop struct {
	id1        string
	id2        int
	sensorType string
	periodMS   atomic.Int64
}

func NewLoop(id1 string, id2 int, sensorType string, hz float64) *Loop {
	l := &Loop{id1: id1, id2: id2, sensorType: sensorType}
	l.SetHz(hz)
	return l
}

func (l *Loop) SetHz(hz float64) {
	if hz <= 0 {
		hz = 0.000001 // practically paused, avoids div-by-zero
	}
	ms := int64(1000.0 / hz)
	if ms <= 0 {
		ms = 1
	}
	l.periodMS.Store(ms)
}

func (l *Loop) SetPeriodMS(ms int64) {
	if ms <= 0 {
		ms = 1
	}
	l.periodMS.Store(ms)
}

func (l *Loop) period() time.Duration {
	ms := l.periodMS.Load()
	if ms <= 0 {
		ms = 1000
	}
	return time.Duration(ms) * time.Millisecond
}

func (l *Loop) Run(ctx context.Context) error {
	// small jitter to avoid lockstep if multiple instances start together
	time.Sleep(time.Duration(rand.Intn(250)) * time.Millisecond)

	t := time.NewTimer(l.period())
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-t.C:
			r := Reading{
				Value:      synth(l.sensorType),
				SensorType: l.sensorType,
				ID1:        l.id1,
				ID2:        l.id2,
				Timestamp:  time.Now().UTC(),
			}
			// Print one JSON per line to stdout (this is our "stream").
			b, _ := json.Marshal(r)
			fmt.Println(string(b))

			t.Reset(l.period())
		}
	}
}

func synth(sensorType string) float64 {
	switch sensorType {
	case "temperature":
		return 18 + rand.Float64()*12 // 18..30
	case "humidity":
		return 30 + rand.Float64()*60 // 30..90
	case "pressure":
		return 980 + rand.Float64()*50 // 980..1030
	default:
		return rand.Float64() * 100
	}
}

// RunWithSink is identical to Run, but calls the provided sink with each Reading.
// It does NOT print to stdout; the caller decides what to do with the reading.
func (l *Loop) RunWithSink(ctx context.Context, sink func(Reading)) error {
	time.Sleep(time.Duration(rand.Intn(250)) * time.Millisecond)

	t := time.NewTimer(l.period())
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-t.C:
			r := Reading{
				Value:      synth(l.sensorType),
				SensorType: l.sensorType,
				ID1:        l.id1,
				ID2:        l.id2,
				Timestamp:  time.Now().UTC(),
			}
			if sink != nil {
				sink(r)
			}
			t.Reset(l.period())
		}
	}
}
