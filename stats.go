package lrucache

import (
	"time"

	"github.com/quipo/statsd"
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
		interval := time.Second * 10
		return statsd.NewStatsdBuffer(interval, statsdclient)
	}
	return nil
}

func (s *Stats) Evict()                    { s.Buffer.Incr("evict", 1) }
func (s *Stats) MemEvicted(evicted uint64) { s.Buffer.Gauge("memEvicted", int64(evicted)) }
