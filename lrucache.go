package lrucache

// Package lrucache implements a least-recently-used cache

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

type (
	GcCallback func()

	LRUCache struct {
		list          *List
		gcFactor      uint32
		gcFactorCfg   uint32
		groups        map[string]*Group
		configuration *Configuration
		getCmdChan    chan GetCmd
		gcCmdChan     chan uint32
		setCmdChan    chan SetCmd
		delCmdChan    chan DelCmd
		dbgCmdChan    chan DbgCmd
	}

	Group struct {
		key   string
		nodes map[string]*Node
	}

	GetCmd struct {
		primaryKey   string
		secondaryKey string
		rtnChan      chan CacheItem
	}

	SetCmd struct {
		primaryKey   string
		secondaryKey string
		item         CacheItem
	}

	DelCmd struct {
		primaryKey   string
		secondaryKey *string
	}

	DbgCmd struct {
		writer   io.Writer
		doneChan chan struct{}
	}

	EvictLog struct {
		Event         string  `json:"event"`
		Source        string  `json:"source"`
		Timestamp     string  `json:"timestamp"`
		Node          string  `json:"node"`
		Group         string  `json:"group"`
		MemoryEvicted float64 `json:"memory_evicted"`
	}
)

func New(configuration *Configuration) *LRUCache {
	c := &LRUCache{
		list:          new(List),
		configuration: configuration,
		groups:        make(map[string]*Group),
		getCmdChan:    make(chan GetCmd),
		gcCmdChan:     make(chan uint32),
		setCmdChan:    make(chan SetCmd),
		delCmdChan:    make(chan DelCmd),
		dbgCmdChan:    make(chan DbgCmd),
	}

	go c.serve()
	go c.gc()

	return c
}

func (c *LRUCache) Get(primaryKey string, secondaryKey string) CacheItem {
	rtnChan := make(chan CacheItem)

	c.getCmdChan <- GetCmd{
		primaryKey:   primaryKey,
		secondaryKey: secondaryKey,
		rtnChan:      rtnChan,
	}
	item := <-rtnChan

	return item
}

func (c *LRUCache) Set(primaryKey string, secondaryKey string, item CacheItem) {
	c.setCmdChan <- SetCmd{
		primaryKey:   primaryKey,
		secondaryKey: secondaryKey,
		item:         item,
	}
}

func (c *LRUCache) serve() {
	for {
		select {
		case getCmd := <-c.getCmdChan:
			group, ok := c.groups[getCmd.primaryKey]
			if ok == false {
				getCmd.rtnChan <- nil
			} else {
				node, ok := group.nodes[getCmd.secondaryKey]
				if ok == false {
					getCmd.rtnChan <- nil
				} else {
					c.list.Promote(node)
					getCmd.rtnChan <- node.item
				}
			}

		case setCmd := <-c.setCmdChan:
			group, ok := c.groups[setCmd.primaryKey]
			if ok == false {
				group = &Group{key: setCmd.primaryKey, nodes: make(map[string]*Node)}
				c.groups[setCmd.primaryKey] = group
			}

			node, ok := group.nodes[setCmd.secondaryKey]
			if ok == false {
				// create new item
				node := &Node{
					item:  setCmd.item,
					group: group,
					key:   setCmd.secondaryKey,
				}
				group.nodes[setCmd.secondaryKey] = node
				c.list.Push(node)
			} else {
				node.item = setCmd.item
				c.list.Promote(node)
			}
			c.purge()

		case gcFactor := <-c.gcCmdChan:
			c.gcFactor = gcFactor

		case delCmd := <-c.delCmdChan:
			group, exists := c.groups[delCmd.primaryKey]
			if exists == true {
				if delCmd.secondaryKey == nil {
					delete(c.groups, delCmd.primaryKey)
					for _, node := range group.nodes {
						c.list.Remove(node)
					}
					continue
				}
				node, exists := group.nodes[*delCmd.secondaryKey]
				if exists == true {
					delete(group.nodes, *delCmd.secondaryKey)
					c.list.Remove(node)
					if len(group.nodes) == 0 {
						delete(c.groups, delCmd.primaryKey)
					}
				}
			}

		case dbgCmd := <-c.dbgCmdChan:
			c.debug(dbgCmd.writer)
			dbgCmd.doneChan <- struct{}{}
		}
	}
}

// time consuming, use carefully.
func (c *LRUCache) Debug(writer io.Writer) {
	doneChan := make(chan struct{})
	c.dbgCmdChan <- DbgCmd{writer: writer, doneChan: doneChan}
	<-doneChan
}

func (c *LRUCache) debug(writer io.Writer) {
	ms := new(runtime.MemStats)
	runtime.ReadMemStats(ms)
	writer.Write([]byte("alloc       : " + strconv.FormatUint(ms.Alloc, 10) + "\n"))
	writer.Write([]byte("heap sys    : " + strconv.FormatUint(ms.HeapSys, 10) + "\n"))
	writer.Write([]byte("heap alloc  : " + strconv.FormatUint(ms.HeapAlloc, 10) + "\n"))
	writer.Write([]byte("total alloc : " + strconv.FormatUint(ms.TotalAlloc, 10) + "\n"))
	writer.Write([]byte("total groups: " + strconv.FormatUint(uint64(len(c.groups)), 10) + "\n"))

	groups := make([]*Group, len(c.groups))
	for _, group := range c.groups {
		groups = append(groups, group)
	}

	newline := []byte("\n")
	tab := []byte("\t")
	for _, group := range c.groups {
		writer.Write([]byte(group.key))
		writer.Write(newline)
		for _, node := range group.nodes {
			writer.Write(tab)
			writer.Write([]byte(node.key))
			writer.Write(tab)
			writer.Write(node.item.Debug())
			writer.Write(newline)
		}
	}
}

func (c *LRUCache) Remove(primaryKey string) bool {
	c.delCmdChan <- DelCmd{primaryKey: primaryKey, secondaryKey: nil}
	return true
}

func (c *LRUCache) RemoveSecondary(primaryKey string, secondaryKey string) bool {
	c.delCmdChan <- DelCmd{primaryKey: primaryKey, secondaryKey: &secondaryKey}
	return true
}

func (c *LRUCache) purge() {
	if c.gcFactor > 0 {
		ms := new(runtime.MemStats)
		runtime.ReadMemStats(ms)
		before := ms.HeapAlloc

		nodes := c.list.Prune(int(c.gcFactor))
		for _, node := range nodes {
			if node == nil {
				break
			}
			runtime.ReadMemStats(ms)
			start := ms.HeapAlloc

			group := node.group
			delete(group.nodes, node.key)
			if len(group.nodes) == 0 {
				delete(c.groups, group.key)
			}

			runtime.ReadMemStats(ms)
			end := ms.HeapAlloc

			logItem := EvictLog{
				Event:         "cacheEvicted",
				Source:        "lrucache",
				Timestamp:     time.Now().Format(time.RFC3339),
				Node:          node.key,
				Group:         group.key,
				MemoryEvicted: math.Abs(float64(start-end) / 1000.0),
			}
			marshalled, _ := json.Marshal(logItem)
			fmt.Println(hydrateString(string(marshalled)))
		}

		runtime.ReadMemStats(ms)
		after := ms.HeapAlloc

		c.configuration.statsd.Evict()
		c.configuration.statsd.MemEvicted(before - after)
	}
}

func (c *LRUCache) setGcFactor(gcFactor uint32) {
	c.gcCmdChan <- gcFactor
}

func (c *LRUCache) ConfigGcFactor(gcFactor uint32) {
	if gcFactor > 0 {
		atomic.StoreUint32(&c.gcFactorCfg, gcFactor)
	}
}

func (c *LRUCache) gc() {
	time.Sleep(30 * time.Second)
	ms := new(runtime.MemStats)
	var gcFactor uint32 = 0
	for {
		runtime.ReadMemStats(ms)

		// update gc factor
		if ms.HeapAlloc < c.configuration.size {
			// stop gc only when cache < 90% of cache limit.
			if gcFactor != 0 && ms.HeapAlloc < uint64(0.9*float64(c.configuration.size)) {
				c.setGcFactor(0)
				gcFactor = 0
			}
		} else {
			if gcFactor == 0 || gcFactor != atomic.LoadUint32(&c.gcFactorCfg) {
				gcFactor = atomic.LoadUint32(&c.gcFactorCfg)
				c.setGcFactor(gcFactor)
			}
		}

		// notify gcing
		if gcFactor != 0 && c.configuration.callback != nil {
			c.configuration.callback()
		}

		time.Sleep(10 * time.Second)
	}
}

func hydrateString(s string) string {
	return strings.Replace(s, "\\u0026", "&", -1)
}
