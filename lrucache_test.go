package lrucache

import (
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/viki-org/gspec"
)

func TestReturnsNilIfPrimaryKeyIsNotInTheCache(t *testing.T) {
	c := New(Configure())
	if c.Get("leto", "") != nil {
		t.Error("expecting nothing to be in the cache")
	}
}

func TestReturnsNilIfSecondaryKeyIsNotInTheCache(t *testing.T) {
	c := New(Configure())
	c.Set("leto", "ghanima", NewItem("SAMPLE BODY FOR TESTING"))
	if c.Get("leto", "duncan") != nil {
		t.Error("expecting nothing to be in the cache")
	}
}

func TestGetReturnsTheItem(t *testing.T) {
	spec := gspec.New(t)
	c := New(Configure())
	item := NewItem("SAMPLE BODY FOR TESTING")
	c.Set("the-p", "the-s", item)
	spec.Expect(c.Get("the-p", "the-s")).ToEqual(item)
}

func TestGetReturnsTheItemWithEmptySecondaryKey(t *testing.T) {
	spec := gspec.New(t)
	c := New(Configure())
	item := NewItem("SAMPLE BODY FOR TESTING")
	c.Set("the-p", "", item)
	spec.Expect(c.Get("the-p", "")).ToEqual(item)
}

func TestGetPromotesTheItemToFrontOfTheCache(t *testing.T) {
	spec := gspec.New(t)
	c := New(Configure())
	item1 := NewItem("SAMPLE BODY FOR TESTING")
	item2 := NewItem("SAMPLE BODY FOR TESTING")
	c.Set("a", "1", item1)
	c.Set("b", "1", item2)
	c.Get("a", "1")
	spec.Expect(c.list.head.item).ToEqual(item1)
}

// ignore, we no longer prune on set, we now prune in a separate timed goroutine
// func TestPrunesWhenAtCapacity(t *testing.T) {
//   spec := gspec.New(t)
//   c := New(5550 + 5550 * int(ITEM_OVERHEAD), false)
//   for i := 0; i < 5550; i++ {
//     c.Set("a", strconv.Itoa(i), &CachedResponse{length: 1})
//   }
//   spec.Expect(len(c.groups)).ToEqual(1)
//   count := 0
//   for node := c.list.head; node != nil; node = node.next { count++ }
//   spec.Expect(count).ToEqual(5550)

//   c.Set("x", "xx", &CachedResponse{length: 1})
//   time.Sleep(100)
//   spec.Expect(len(c.groups)).ToEqual(2)
//   count = 0
//   for node := c.list.head; node != nil; node = node.next { count++ }
//   spec.Expect(count).ToEqual(551)
//   spec.Expect(c.list.tail.group.key).ToEqual("a")
//   spec.Expect(c.list.tail.key).ToEqual("5000")
// }

func TestRemovesAllSecondaryItemsFromTheCache(t *testing.T) {
	spec := gspec.New(t)
	c := New(Configure().Size(550))
	item1 := NewItem("SAMPLE BODY FOR TESTING")
	c.Set("a", "1", item1)
	c.Set("b", "2", NewItem("SAMPLE BODY FOR TESTING"))
	c.Set("b", "3", NewItem("SAMPLE BODY FOR TESTING"))
	c.Remove("b")
	spec.Expect(c.Get("a", "1")).ToEqual(item1)
	spec.Expect(len(c.groups)).ToEqual(1) // b has to be removed since there'2 only 1 item, and we know, from the above, that it's a
}

func TestHandlesRemovalOfAnInvalidPrimaryKey(t *testing.T) {
	spec := gspec.New(t)
	c := New(Configure().Size(550))
	item1 := NewItem("SAMPLE BODY FOR TESTING")
	c.Set("a", "1", item1)
	c.Remove("b")
	spec.Expect(c.Get("a", "1")).ToEqual(item1)
	spec.Expect(len(c.groups)).ToEqual(1) // b has to be removed since there'2 only 1 item, and we know, from the above, that it's a
}

func TestRemovesAnIndividualSecondaryItemFromTheCache(t *testing.T) {
	spec := gspec.New(t)
	c := New(Configure().Size(550))
	item1 := NewItem("SAMPLE BODY FOR TESTING")
	item2 := NewItem("SAMPLE BODY FOR TESTING")
	c.Set("a", "1", item1)
	c.Set("b", "2", item2)
	c.Set("b", "3", NewItem("SAMPLE BODY FOR TESTING"))
	c.RemoveSecondary("b", "3")
	spec.Expect(c.Get("a", "1")).ToEqual(item1)
	spec.Expect(c.Get("b", "2")).ToEqual(item2)
	spec.Expect(c.Get("b", "3")).ToEqual(nil)
	spec.Expect(len(c.groups)).ToEqual(2)
}

func TestHandlesRemovalOfAnInvalidSecondaryKey(t *testing.T) {
	spec := gspec.New(t)
	c := New(Configure().Size(550))
	item1 := NewItem("SAMPLE BODY FOR TESTING")
	item2 := NewItem("SAMPLE BODY FOR TESTING")
	c.Set("a", "1", item1)
	c.Set("b", "2", item2)
	c.Set("b", "3", NewItem("SAMPLE BODY FOR TESTING"))
	c.RemoveSecondary("b", "c")
	spec.Expect(len(c.groups["a"].nodes)).ToEqual(1)
	spec.Expect(len(c.groups["b"].nodes)).ToEqual(2)
	spec.Expect(len(c.groups)).ToEqual(2)
}

func TestConcurrencyStress(t *testing.T) {
	fmt.Printf("%v\n", time.Now())
	c := New(Configure().Size(550))
	var wg sync.WaitGroup
	for i := 1; i < 1000; i++ {
		wg.Add(1)
		go func() {
			for j := 1; j < 2; j++ {
				for _, cp := range "helloviki" {
					p := string(cp)
					for _, cs := range "helloviki" {
						item := NewItem("test string")
						s := string(cs)
						c.Set(p, s, item)
						c.Get(p, s)
						c.RemoveSecondary(p, s)
						// c.Remove(p)
						if cs == 'v' {
							c.gcFactor = 1
						} else {
							c.gcFactor = 0
						}
						c.Set(p, s, item)
					}
				}
			}
			wg.Done()
		}()
	}

	wg.Wait()
	c.Debug(os.Stdout)
	fmt.Printf("%v\n", time.Now())
}

type ItemToCache struct {
	body   string
	length int
}

func (i *ItemToCache) Debug() []byte {
	return []byte("")
}

func (i *ItemToCache) Expires() time.Time {
	return time.Now()
}

func NewItem(body string) *ItemToCache {
	return &ItemToCache{
		body:   body,
		length: len(body),
	}
}
