package lrucache

type Configuration struct {
  size uint64
  callback GcCallback
  itemsToPrune int
}

func Configure() *Configuration {
  return &Configuration{
    callback: nil,
    itemsToPrune: 10000,
    size: 50 * 1024 * 1024 * 1024,
  }
}

func (c *Configuration) Size(size uint64) (*Configuration) {
  c.size = size
  return c
}

func (c *Configuration) SizeInt(size int) (*Configuration) {
  c.size = uint64(size)
  return c
}

func (c *Configuration) Callback(callback GcCallback) (*Configuration) {
  c.callback = callback
  return c
}

func (c *Configuration) ItemsToPrune(itemsToPrune int) (*Configuration) {
  c.itemsToPrune = itemsToPrune
  return c
}
