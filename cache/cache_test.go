package cache

import (
	"fmt"
	"log"
	"time"

	lru "github.com/beatlabs/patron/cache/lru"
)

func Example() {
	// You can use any of the available implementations of the `Cache` interface,
	// or provide your own. This example uses the `lru` implementation.
	// All actions return an error for proper handling, which is out of the scope
	// of this example. Let's create a cache that can hold 32 items.
	c, err := lru.Create(32)
	if err != nil {
		log.Fatal(err)
	}

	k, v := "foo", "bar"

	// Set a key-value pair in the cache.
	_ = c.Set(k, v)

	// Check whether the cache contains a specific key.
	ok, _ := c.Contains(k)
	fmt.Println(ok)

	ok, _ = c.Contains("patron")
	fmt.Println(ok)

	// Retrieve the cached value of some key. This operation
	// also reports if the key exists in the cache.
	val, ok, _ := c.Get(k)
	fmt.Println(val, ok)

	// Remove a key from the cache.
	_ = c.Remove(k)
	ok, _ = c.Contains(k)
	fmt.Println(ok)

	// You can set a key with a preset expiry/TTL.
	_ = c.SetTTL(k, v, 1*time.Hour)

	// And finally, you can nuke the whole cache, if needed.
	_ = c.Purge()

	// Output:
	// true
	// false
	// bar true
	// false
}
