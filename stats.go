package lrucache

import (
	"log"
	"time"

	"github.com/quipo/statsd"
)

var (
	defaultInterval = time.Second * 10
)

type Stats struct {
	Buffer *statsd.StatsdBuffer
}

func NewStats(address, prefix string) *Stats {
	return &Stats{
		Buffer: NewStatsdBuffer(address, prefix),
	}
}

func NewStatsdBuffer(address, prefix string) *statsd.StatsdBuffer {
	statsdclient := statsd.NewStatsdClient(address, prefix)
	if err := statsdclient.CreateSocket(); err != nil {
		log.Printf("Error creating statsd client: %v\n", err.Error())
		return nil
	}

	return statsd.NewStatsdBuffer(defaultInterval, statsdclient)
}

func (s *Stats) Evict() {
	if s.Buffer == nil {
		return
	}
	s.Buffer.Incr("evict", 1)
}
func (s *Stats) MemEvicted(evicted uint64) {
	if s.Buffer == nil {
		return
	}
	s.Buffer.Gauge("memEvicted", int64(evicted))
}
