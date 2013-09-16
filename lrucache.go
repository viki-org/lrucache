package lrucache
// Package lrucache implements a least-recently-used cache

import (
  "io"
  "time"
  "sync"
  "runtime"
  "strconv"
  "sync/atomic"
)

type GcCallback func()

type LRUCache struct {
  list *List
  size uint64
  sync.RWMutex
  groups map[string]*Group
}

type Group struct {
  key string
  sync.RWMutex
  nodes map[string]*Node
}

func New(size int, gccallback GcCallback) *LRUCache {
  c := &LRUCache {
    size: uint64(size),
    list: new(List),
    groups: make(map[string]*Group, 50000),
  }
  go c.gc(gccallback)
  return c
}

func (c *LRUCache) Get(primaryKey string, secondaryKey string) CacheItem {
  group, node := c.getNode(primaryKey, secondaryKey)
  if node == nil { return nil }
  c.promote(group, node)
  return node.item
}

func (c *LRUCache) Set(primaryKey string, secondaryKey string, item CacheItem) {
  group, existing  := c.getNode(primaryKey, secondaryKey)
  if existing != nil {
    group.Lock()
    existing.item = item
    group.Unlock()
    c.promote(group, existing)
  } else {
    if group == nil {
      c.Lock()
      group = c.groups[primaryKey]
      if group == nil {
        group = &Group{key: primaryKey, nodes: make(map[string]*Node, 5000),}
        c.groups[primaryKey] = group
      }
      c.Unlock()
    }
    node := &Node{
      item: item,
      group: group,
      key: secondaryKey,
      promotable: time.Now().Add(time.Minute * 10),
    }
    group.Lock()
    group.nodes[secondaryKey] = node
    group.Unlock()
    c.list.Push(node)
  }
}

func (c *LRUCache) Debug(writer io.Writer) {
  ms := new(runtime.MemStats)
  runtime.ReadMemStats(ms)
  writer.Write([]byte("alloc      : " + strconv.FormatUint(ms.Alloc, 10) + "\n"))
  writer.Write([]byte("heap sys   : " + strconv.FormatUint(ms.HeapSys, 10) + "\n"))
  writer.Write([]byte("heap alloc : " + strconv.FormatUint(ms.HeapAlloc, 10) + "\n"))
  writer.Write([]byte("total alloc: " + strconv.FormatUint(ms.TotalAlloc, 10) + "\n"))


  c.RLock()
  groups := make([]*Group, len(c.groups))
  for _, group := range c.groups {
    groups = append(groups, group)
  }
  defer c.RUnlock()

  newline := []byte("\n")
  tab := []byte("\t")
  for _, group := range c.groups {
    group.RLock()
    writer.Write([]byte(group.key))
    writer.Write(newline)
    for _, node := range group.nodes {
     writer.Write(tab)
     writer.Write([]byte(node.key))
     writer.Write(tab)
     writer.Write(node.item.Debug())
     writer.Write(newline)
    }
    group.RUnlock()
  }
}

func (c* LRUCache) UpdateCapacity(size int) {
  c.Lock()
  c.size = uint64(size)
  c.Unlock()
}

func (c *LRUCache) Remove(primaryKey string) bool {
  c.RLock()
  group, exists := c.groups[primaryKey]
  c.RUnlock()
  if exists == false { return false }

  c.Lock()
  delete(c.groups, primaryKey)
  c.Unlock()

  for _, node := range group.nodes {
    c.list.Remove(node)
  }
  return true
}

func (c *LRUCache) RemoveSecondary(primaryKey string, secondaryKey string) bool {
  c.RLock()
  group, exists := c.groups[primaryKey]
  c.RUnlock()
  if exists == false { return false }

  group.Lock()
  defer group.Unlock()
  if _, exists = group.nodes[secondaryKey]; exists == false { return false }
  delete(group.nodes, secondaryKey)
  if len(group.nodes) == 0 {
    c.Lock()
    delete(c.groups, primaryKey)
    c.Unlock()
  }
  return true
}

func (c *LRUCache) getNode(primaryKey string, secondaryKey string) (*Group, *Node) {
  c.RLock()
  group, ok := c.groups[primaryKey]
  c.RUnlock()
  if ok == false { return nil, nil }

  group.RLock()
  node, _ := group.nodes[secondaryKey]
  group.RUnlock()
  return group, node
}

func (c *LRUCache) promote(group *Group, node *Node) {
  now := time.Now()
  group.RLock()
  promotable := node.promotable
  group.RUnlock()
  if now.Before(promotable) { return }

  group.Lock()
  if now.After(node.promotable) {
    c.list.Promote(node)
    node.promotable = now.Add(time.Minute * 10)
  }
  group.Unlock()
}

func (c *LRUCache) gc(callback GcCallback) {
  time.Sleep(30 * time.Second)
  ms := new(runtime.MemStats)
  for {
    runtime.ReadMemStats(ms)
    if ms.HeapAlloc < atomic.LoadUint64(&c.size) {
      time.Sleep(30 * time.Second)
      continue
    }
    nodes := c.list.Prune(10000)
    for _, node := range nodes {
      if node == nil { break }
      group := node.group
      group.Lock()
      delete(group.nodes, node.key)
      if len(group.nodes) == 0 {
        c.Lock()
        delete(c.groups, group.key)
        c.Unlock()
      }
      group.Unlock()
    }
    if callback != nil { callback() }
  }
}
