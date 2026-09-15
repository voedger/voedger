# Pool

Package `pool` manages reusable Go objects. It initializes an object
on every borrow, cleans it before reuse, protects against duplicate
releases, and can tie a borrowed object's lifetime to an owner.

There are two ways to borrow an object: without an owner by using
`Get()`, or with an owner by using `GetOwned()`.

## Borrowing without an owner

A plain `sync.Pool` stores reusable objects but does not manage their
lifecycle or track their use. Each pooled type needs custom borrow,
release, counting, and diagnostic code.

<details>
<summary>Without pool</summary>

```go
package main

import (
	"sync"
	"sync/atomic"
)

type object struct {
	value string
}

var (
	objects = sync.Pool{
		New: func() any { return &object{} },
	}
	objectsInUse atomic.Int64
)

func getObject() *object {
	o := objects.Get().(*object)
	o.value = "initialized" // Initialize on every borrow.
	objectsInUse.Add(1)     // Update on every borrow path.
	return o
}

func releaseObject(o *object) {
	o.value = ""         // Clean up before every pool return.
	objectsInUse.Add(-1) // Update on every release path.
	objects.Put(o)       // No duplicate-release protection.
}

func main() {
	o := getObject()
	releaseObject(o)
	releaseObject(o) // BUG: duplicate return corrupts pool and count.

	// Leak call sites require a separate stack-trace registry.
}
```

A complete implementation also needs a thread-safe release guard on
every object and synchronized storage for borrow stack traces. Omitting
any of that bookkeeping can corrupt the pool or hide leaks.

</details>

<details>
<summary>With pool</summary>

Embed `pool.IReleaser` in the pooled type and assign the releaser
supplied to the `NewPool` factory. The optional `Init()` hook runs on
every `Get()`; the optional `Cleanup()` hook runs on `Release()`, before
the object is returned to the pool.

```go
package main

import (
	"fmt"
	"os"

	"github.com/voedger/voedger/pkg/goutils/pool"
)

type object struct {
	// Embed this interface in every pooled type.
	pool.IReleaser

	value string
}

// Init runs automatically on every borrow.
func (o *object) Init() {
	o.value = "initialized"
}

// Cleanup runs before the object returns to the pool.
func (o *object) Cleanup() {
	o.value = "" // Clear application state before reuse.
}

var objects = pool.NewPool(func(releaser pool.IReleaser) *object {
	return &object{
		// Store the implementation supplied by the pool.
		IReleaser: releaser,
	}
})

func main() {
	// Record where outstanding objects were borrowed.
	pool.SetDebug(true)
	defer pool.SetDebug(false)

	o := objects.Get()                  // Runs Init automatically.
	fmt.Println(o.value)                // Set by Init: "initialized"
	fmt.Println(pool.GetObjectsInUse()) // One object is in use.
	pool.PrintNonReleased(os.Stdout)     // Reports this Get call site.

	o.Release()                         // Runs Cleanup, then returns o.
	fmt.Println(pool.GetObjectsInUse()) // No objects are in use.
	pool.PrintNonReleased(os.Stdout)     // Reports nothing.

	// Calling o.Release() again panics: "already released".
}
```

</details>

After `Release()`, neither the object nor its fields may be accessed. A
second release is rejected before `Cleanup()` or the pool return can
happen again.

`GetObjectsInUse()` always counts outstanding objects across all
registered pools. Debug mode additionally records the call stacks of
outstanding `Get()` calls so that `PrintNonReleased()` can report where
they occurred. Enable it before borrowing; it adds runtime overhead.
Disabling debug mode stops recording new borrows, while existing records
remain until their objects are released.

Use `NewPoolStub()` instead of `NewPool()` when investigating
reuse-related problems. A stub creates a fresh object for every borrow
while retaining the same initialization, cleanup, ownership, release
checks, counters, and diagnostics.

If `Init()` panics, the panic propagates and the object remains counted.
A standalone borrow made in debug mode also remains in the leak report.
The pool cannot automatically clean up partial initialization.

See the [standalone object tests](impl_test.go#L41).

## Borrowing with an owner

With `sync.Pool`, ownership is only a convention. Developers must add
and maintain ownership state, block direct release of owned items, and
return every owned object when its owner is released.

<details>
<summary>Without pool</summary>

```go
package main

import "sync"

type item struct{}

type owner struct {
	item *item
}

var items = sync.Pool{
	New: func() any { return &item{} },
}

var owners = sync.Pool{
	New: func() any { return &owner{} },
}

func getItem() *item {
	return items.Get().(*item)
}

func releaseItem(i *item) {
	items.Put(i) // No ownership or duplicate-release checks.
}

func getOwner() *owner {
	o := owners.Get().(*owner)
	o.item = getItem() // Ownership exists only by convention.
	return o
}

func releaseOwner(o *owner) {
	releaseItem(o.item) // Easy to omit or call twice.
	o.item = nil
	owners.Put(o)
}

func main() {
	standaloneItem := getItem()
	releaseItem(standaloneItem) // Valid for a standalone item.

	o := getOwner()
	releaseItem(o.item) // BUG: nothing prevents an early release.
	releaseOwner(o)     // BUG: returns the same item again.
}
```

Preventing these errors manually requires release guards on both types,
an internal release path for owned items, and an ownership list for each
owner. Those mechanisms must also handle nested ownership and concurrent
release attempts.

</details>

<details>
<summary>With pool</summary>

Use `GetOwned(owner)` when a pooled object's lifetime must not be
shorter than its owner's. An owned object cannot be released directly;
releasing the owner automatically cleans and releases every object it
owns, including owned descendants. The same object type and pool can be
used for both standalone and owned borrows.

```go
package main

import (
	"fmt"

	"github.com/voedger/voedger/pkg/goutils/pool"
)

// An item can be borrowed either standalone or owned.
type item struct {
	pool.IReleaser
}

var items = pool.NewPool(func(releaser pool.IReleaser) *item {
	return &item{IReleaser: releaser}
})

type owner struct {
	pool.IReleaser
	item *item
}

func (o *owner) Init() {
	// Tie the item's lifetime to o.
	o.item = items.GetOwned(o)
}

func (o *owner) Cleanup() {
	// The pool releases the item after this method.
	o.item = nil
}

var owners = pool.NewPool(func(releaser pool.IReleaser) *owner {
	return &owner{IReleaser: releaser}
})

func main() {
	// Borrow the item without an owner.
	standaloneItem := items.Get()
	fmt.Println(standaloneItem.IsOwned()) // false: not owned
	standaloneItem.Release()              // Direct release succeeds.
	fmt.Println(pool.GetObjectsInUse())   // No objects are in use.

	// Borrow the same item type with an owner.
	o := owners.Get()                    // Borrows owner and owned item.
	fmt.Println(pool.GetObjectsInUse())  // Two objects are in use.
	fmt.Println(o.IsOwned())             // false: not owned
	fmt.Println(o.item.IsOwned())        // true: owned by o

	// o.item.Release() panics: "must be released by owner".

	o.Release()                         // Releases the item and owner.
	fmt.Println(pool.GetObjectsInUse()) // No objects are in use.
}
```

</details>

Do not release owned objects from the owner's `Cleanup()` hook. The hook
runs before the pool walks the ownership chain and releases the owned
objects. As with any released object, the owner and its owned items must
not be accessed after the owner's `Release()` call.

Owned borrows contribute to `GetObjectsInUse()`, but only standalone
`Get()` calls have their own stack traces in `PrintNonReleased()`.

See the [owned object tests](owned_test.go#L66).
