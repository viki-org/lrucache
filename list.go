package lrucache

type List struct {
	head *Node
	tail *Node
}

type Node struct {
	group *Group
	key   string
	next  *Node
	prev  *Node
	item  CacheItem
}

// Add a node to the list.
func (l *List) Push(node *Node) {
	node.next = l.head
	if l.head != nil {
		l.head.prev = node
	}
	l.head = node
	if l.tail == nil {
		l.tail = node
	}
}

// Remove a node from list.
func (l *List) Remove(node *Node) {
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
}

// Promote a node already in list.
func (l *List) Promote(node *Node) {
	// remove
	if node.prev == nil {
		// already head, return.
		return
	} else {
		node.prev.next = node.next
	}

	if l.tail == node {
		l.tail = node.prev
	} else {
		node.next.prev = node.prev
	}

	// push
	node.next = l.head
	node.prev = nil
	l.head.prev = node
	l.head = node
}

func (l *List) Prune(count int) (nodes []*Node) {
	nodes = make([]*Node, count)
	for i := 0; i < count; i++ {
		node := l.tail
		if node == nil {
			return
		}
		nodes[i] = node
		l.tail = node.prev
		if node.prev != nil {
			node.prev.next = nil
		} else {
			l.head = nil
			l.tail = nil
			return
		}
		node.prev = nil
		node.next = nil
	}
	return
}
