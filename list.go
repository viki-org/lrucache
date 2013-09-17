package lrucache

import (
  "time"
  "sync"
)

type List struct {
  head *Node
  tail *Node
  sync.Mutex
}

type Node struct {
  group *Group
  key string
  next *Node
  prev *Node
  item CacheItem
  promotable time.Time
}

func (l *List) Push(node *Node) {
  l.Lock()
  node.next = l.head
  if l.head != nil { l.head.prev = node }
  l.head = node
  if l.tail == nil { l.tail = node }
  l.Unlock()
}

func (l *List) Remove(node *Node) {
  l.Lock()
  if node.prev == nil {
    l.head = node.next
  } else {
    node.prev.next = node.next
  }
  if node.next == nil {
    l.tail = node.prev
  } else {
    node.next.prev = node.prev
  }
  l.Unlock()
}

func (l *List) Promote(node *Node) {
  if l.head == node { return }
  if node.prev == nil { //the node was disconnected
    l.Push(node)
    return
  }
  l.Lock()
  if l.tail == node {
    l.tail = node.prev
  } else {
    node.next.prev = node.prev
  }
  node.prev.next = node.next
  node.next = l.head
  l.head = node
  node.next.prev = node
  l.Unlock()
}

func (l *List) Prune(count int) (nodes []*Node) {
  nodes = make([]*Node, count)
  l.Lock()
  defer l.Unlock()
  for i := 0; i < count; i++ {
    node := l.tail
    if node == nil { return }
    nodes[i] = node
    l.tail = node.prev
    if node.prev != nil {
      node.prev.next = nil
    } else {
      l.head = nil
      l.tail = nil
      return
    }
  }
  return
}
