package lrucache

import (
  "testing"
  "apiproxy/tests"
)

func TestAddsNodesToTheHead(t *testing.T) {
  l := new(List)

  three := new(Node)
  l.Push(three)
  assertList(t, l, three)

  two := new(Node)
  l.Push(two)
  assertList(t, l, two, three)

  one := new(Node)
  l.Push(one)
  assertList(t, l, one, two, three)
}

func TestPromotesFromHead(t *testing.T) {
  list, one, two, three := sampleList()
  list.Promote(one)
  assertList(t, list, one, two, three)
}

func TestPromotesFromTail(t *testing.T) {
  list, one, two, three := sampleList()
  list.Promote(three)
  assertList(t, list, three, one, two)
}

func TestPromotesFromTheMiddle(t *testing.T) {
  list, one, two, three := sampleList()
  list.Promote(two)
  assertList(t, list, two, one, three)
}

func TestPrunesTheList1(t *testing.T) {
  spec := tests.Spec(t)
  l, one, two, three := sampleList()
  items := l.Prune(1)
  assertList(t, l, one, two)
  spec.Expect(items[0]).ToEqual(three)
  spec.Expect(len(items)).ToEqual(1)
}

func TestPrunesTheList2(t *testing.T) {
  spec := tests.Spec(t)
  l, one, two, three := sampleList()
  items := l.Prune(2)
  assertList(t, l, one)
  spec.Expect(items[0]).ToEqual(three)
  spec.Expect(items[1]).ToEqual(two)
  spec.Expect(len(items)).ToEqual(2)
}

func TestPrunesAllNodes(t *testing.T) {
  l, _, _, _ := sampleList()
  l.Prune(3)
  if l.head != nil { t.Error("head should be nil") }
  if l.tail != nil { t.Error("tail should be nil") }
}

func TestPrunesAllNodesAndMore(t *testing.T) {
  l, _, _, _ := sampleList()
  l.Prune(5)
  if l.head != nil { t.Error("head should be nil") }
  if l.tail != nil { t.Error("tail should be nil") }
}

func TestRemoveNodeFromHead(t *testing.T) {
  l, one, two, three := sampleList()
  l.Remove(one)
  assertList(t, l, two, three)
}

func TestRemoveNodeFromTail(t *testing.T) {
  l, one, two, three := sampleList()
  l.Remove(three)
  assertList(t, l, one, two)
}

func TestRemoveNodeFromMiddle(t *testing.T) {
  l, one, two, three := sampleList()
  l.Remove(two)
  assertList(t, l, one, three)
}

func assertList(t *testing.T, l *List, items ... *Node) {
  if l.head != items[0] {  }
  if l.tail != items[len(items)-1] { t.Errorf("expected tail to equal %+v,got %+v", items[len(items)-1], l.tail) }

  current := l.head
  for index, item := range items {
    if current != item {
      t.Errorf("expected item at position %d to equal %+v,got %+v", index, item, current)
    }
    if index > 0 && current.prev != items[index-1] {
      t.Errorf("expected prev at at position %d to equal %+v,got %+v", index, items[index-1], current.prev)
    }
    current = current.next
  }
  if current != nil {
    t.Error("Tail's next should be nil")
  }
}

func sampleList() (*List, *Node, *Node, *Node) {
  l := new(List)
  three := new(Node)
  two := new(Node)
  one := new(Node)
  l.Push(three)
  l.Push(two)
  l.Push(one)
  return l, one, two, three
}
