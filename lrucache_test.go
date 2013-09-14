package lrucache

import (
  "time"
  "testing"
  "github.com/viki-org/gspec"
)

func TestReturnsNilIfPrimaryKeyIsNotInTheCache(t *testing.T) {
  c := New(10000)
  if c.Get("leto", "") != nil { t.Error("expecting nothing to be in the cache") }
}

func TestReturnsNilIfSecondaryKeyIsNotInTheCache(t *testing.T) {
  c := New(10000)
  c.Set("leto", "ghanima", NewItem("SAMPLE BODY FOR TESTING"))
  if c.Get("leto", "duncan") != nil { t.Error("expecting nothing to be in the cache") }
}

func TestGetReturnsTheItem(t *testing.T) {
  spec := gspec.New(t)
  c := New(10000)
  item := NewItem("SAMPLE BODY FOR TESTING")
  c.Set("the-p", "the-s", item)
  spec.Expect(c.Get("the-p", "the-s")).ToEqual(item)
}

func TestGetReturnsTheItemWithEmptySecondaryKey(t *testing.T) {
  spec := gspec.New(t)
  c := New(10000)
  item := NewItem("SAMPLE BODY FOR TESTING")
  c.Set("the-p", "", item)
  spec.Expect(c.Get("the-p", "")).ToEqual(item)
}

func TestGetPromotesTheItemToFrontOfTheCache(t *testing.T) {
  spec := gspec.New(t)
  c := New(10000)
  item1 := NewItem("SAMPLE BODY FOR TESTING")
  item2 := NewItem("SAMPLE BODY FOR TESTING")
  c.Set("a", "1", item1)
  c.list.head.promotable = time.Now().Add(time.Minute * -6)
  c.Set("b", "1", item2)
  c.Get("a", "1")
  spec.Expect(c.list.head.item).ToEqual(item1)
}

func TestGetDoesNotPromoteRecentlyPromotedItem(t *testing.T) {
  spec := gspec.New(t)
  c := New(10000)
  item1 := NewItem("SAMPLE BODY FOR TESTING")
  item2 := NewItem("SAMPLE BODY FOR TESTING")
  c.Set("a", "1", item1)
  c.list.head.promotable = time.Now().Add(time.Minute)
  c.Set("b", "1", item2)
  c.Get("a", "1")
  spec.Expect(c.list.head.item).ToEqual(item2)
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
  c := New(550)
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
  c := New(550)
  item1 := NewItem("SAMPLE BODY FOR TESTING")
  c.Set("a", "1", item1)
  c.Remove("b")
  spec.Expect(c.Get("a", "1")).ToEqual(item1)
  spec.Expect(len(c.groups)).ToEqual(1) // b has to be removed since there'2 only 1 item, and we know, from the above, that it's a
}

func TestRemovesAnIndividualSecondaryItemFromTheCache(t *testing.T) {
  spec := gspec.New(t)
  c := New(550)
  item1 := NewItem("SAMPLE BODY FOR TESTING")
  item2 :=  NewItem("SAMPLE BODY FOR TESTING")
  c.Set("a", "1", item1)
  c.Set("b", "2", item2)
  c.Set("b", "3", NewItem("SAMPLE BODY FOR TESTING"))
  c.RemoveSecondary("b", "3")
  spec.Expect(c.Get("a", "1")).ToEqual(item1)
  spec.Expect(c.Get("b", "2")).ToEqual(item2)
  spec.Expect(c.Get("b", "3")).ToEqual(nil)
  spec.Expect(len(c.groups)).ToEqual(2)
}

func TestHandelsRemovalOfAnInvalidSecondaryKey(t *testing.T) {
  spec := gspec.New(t)
  c := New(550)
  item1 := NewItem("SAMPLE BODY FOR TESTING")
  item2 :=  NewItem("SAMPLE BODY FOR TESTING")
  c.Set("a", "1", item1)
  c.Set("b", "2", item2)
  c.Set("b", "3", NewItem("SAMPLE BODY FOR TESTING"))
  c.RemoveSecondary("b", "c")
  spec.Expect(len(c.groups["a"].nodes)).ToEqual(1)
  spec.Expect(len(c.groups["b"].nodes)).ToEqual(2)
  spec.Expect(len(c.groups)).ToEqual(2)
}

func TestUpdateCapacity(t *testing.T) {
  spec := gspec.New(t)
  c := New(10000)
  item := NewItem("TestUpdateCapacity")
  item.length = 2500
  c.Set("abc", "1", item)
  // -4 for len(primaryKey) + len(secondaryKey)
  spec.Expect(c.totalCapacity).ToEqual(10000)
  spec.Expect(c.capacity).ToEqual(int64(7500 - ITEM_OVERHEAD - 4))

  c.UpdateCapacity(7500)
  spec.Expect(c.totalCapacity).ToEqual(7500)
  spec.Expect(c.capacity).ToEqual(int64(5000) - ITEM_OVERHEAD - 4)
}


type ItemToCache struct {
  body string
  length int
}

func (i *ItemToCache) Size() int64 {
  return int64(i.length)
}

func (i *ItemToCache) Expires() time.Time {
  return time.Now()
}

func NewItem(body string) *ItemToCache {
  return &ItemToCache {
    body: body,
    length: len(body),
  }
}
