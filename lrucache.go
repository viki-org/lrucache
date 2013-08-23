package lrucache
// Package lrucache implements a least-recently-used cache

import (
  "time"
  "sync"
  "sync/atomic"
)

const ITEM_OVERHEAD = int64(350) // got this from inspecting memory usage and rounding up to 5x64

type LRUCache struct {
  totalCapacity int
  capacity int64

  lock sync.RWMutex
  groups map[string]*Group
  list *List
}

type Group struct {
  key string
  size int64
  lock sync.RWMutex
  nodes map[string]*Node
}

func New(size int) *LRUCache {
  c := &LRUCache {
    capacity: int64(size),
    totalCapacity: size,
    list: new(List),
    groups: make(map[string]*Group, 50000),
  }
  go c.gc()
  return c
}

func (c *LRUCache) Get(primaryKey string, secondaryKey string) CacheItem {
  group, node := c.getNode(primaryKey, secondaryKey)
  if node == nil { return nil }
  c.promote(group, node)
  return node.item
}

func (c *LRUCache) Set(primaryKey string, secondaryKey string, item CacheItem) {
  size := item.Size() + ITEM_OVERHEAD
  group, existing  := c.getNode(primaryKey, secondaryKey)
  if existing != nil {
    existingSize := existing.item.Size() + ITEM_OVERHEAD
    group.lock.Lock()
    group.size += existingSize - size
    existing.item = item
    group.lock.Unlock()
    c.promote(group, existing)
    atomic.AddInt64(&c.capacity, existingSize + size)
  } else {
    if group == nil {
      c.lock.Lock()
      group = c.groups[primaryKey]
      if group == nil {
        group = &Group{key: primaryKey, nodes: make(map[string]*Node, 5000),}
        c.groups[primaryKey] = group
      }
      c.lock.Unlock()
    }
    node := &Node{
      group: group,
      key: secondaryKey,
      item: item,
      promotable: time.Now().Add(time.Minute * 10),
    }
    group.lock.Lock()
    group.nodes[secondaryKey] = node
    group.size += size
    group.lock.Unlock()
    c.list.Push(node)
    atomic.AddInt64(&c.capacity, -size)
  }
}

func (c* LRUCache) UpdateCapacity(capacity int) {
  c.lock.Lock()
  c.capacity += int64(capacity) - int64(c.totalCapacity)
  c.totalCapacity = capacity
  c.lock.Unlock()
}

func (c *LRUCache) Remove(primaryKey string) bool {
  c.lock.RLock()
  group, exists := c.groups[primaryKey]
  c.lock.RUnlock()
  if exists == false { return false }

  c.lock.Lock()
  delete(c.groups, primaryKey)
  c.capacity += group.size
  c.lock.Unlock()

  for _, node := range group.nodes {
    c.list.Remove(node)
  }
  return true
}


func (c *LRUCache) RemoveSecondary(primaryKey string, secondaryKey string) bool {
  c.lock.RLock()
  group, exists := c.groups[primaryKey]
  c.lock.RUnlock()
  if exists == false { return false }

  group.lock.Lock()
  defer group.lock.Unlock()
  node, exists := group.nodes[secondaryKey]
  if exists == false { return false }
  group.size -= node.item.Size()
  delete(group.nodes, secondaryKey)
  return true
}

func (c *LRUCache) getNode(primaryKey string, secondaryKey string) (*Group, *Node) {
  c.lock.RLock()
  group, ok := c.groups[primaryKey]
  c.lock.RUnlock()
  if ok == false { return nil, nil }

  group.lock.RLock()
  node, _ := group.nodes[secondaryKey]
  group.lock.RUnlock()
  return group, node
}

func (c *LRUCache) promote(group *Group, node *Node) {
  now := time.Now()
  group.lock.RLock()
  promotable := node.promotable
  group.lock.RUnlock()
  if now.Before(promotable) { return }

  group.lock.Lock()
  if now.After(node.promotable) {
    c.list.Promote(node)
    node.promotable = now.Add(time.Minute * 10)
  }
  group.lock.Unlock()
}

// GC won't remove empty groups
// This makes it possible to avoid ever locking the cache when GCing
// Plus, a group has little overhead, and seen once, it'll likely be seen again
// Thus avoid unecessary allocations
// Despite all these reasons, I think I'm just being lazy
func (c *LRUCache) gc() {
  for {
    if atomic.LoadInt64(&c.capacity) > 0 {
      time.Sleep(5 * time.Second)
      continue
    }
    nodes := c.list.Prune(10000)
    var freed int64
    for _, node := range nodes {
      if node == nil { break }
      size := node.item.Size() + ITEM_OVERHEAD
      group := node.group
      group.lock.Lock()
      delete(group.nodes, node.key)
      group.size -= size
      group.lock.Unlock()
      freed += size
    }
    atomic.AddInt64(&c.capacity, freed)
  }
}
