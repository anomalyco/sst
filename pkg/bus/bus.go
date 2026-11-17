package bus

import (
	"log/slog"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	// Buffer for a typed subscriber. These carry control flow, so the buffer is
	// sized to absorb any realistic burst; past it the consumer is not coming
	// back and the backlog is only costing memory. Each slot is an interface
	// header, so this is ~256KB per subscriber, paid at Subscribe.
	typedBuffer = 1 << 14

	// Buffer for a firehose subscriber. Wider fan-out, cheaper to lose: these
	// feed the console and log streams.
	firehoseBuffer = 10_000
)

var (
	bus = &EventBus{
		subscribers: make(map[reflect.Type][]*subscriber),
		all:         make([]*subscriber, 0),
	}
	dropped  atomic.Uint64
	lastWarn atomic.Int64
)

type EventBus struct {
	subscribers map[reflect.Type][]*subscriber
	mu          sync.RWMutex
	all         []*subscriber
}

// subscriber owns one consumer's channel and its delivery policy, which is
// just how much backlog it is allowed before events are dropped.
//
// Delivery never blocks the publisher. Blocking would hold the bus read lock
// for as long as the consumer is wedged, which blocks Subscribe and
// Unsubscribe and — because a pending writer excludes new readers — every
// other publisher with them.
type subscriber struct {
	label   string
	ch      chan interface{}
	lossy   bool
	dropped atomic.Uint64
}

func newSubscriber(label string, lossy bool) *subscriber {
	size := typedBuffer
	if lossy {
		size = firehoseBuffer
	}
	return &subscriber{
		label: label,
		lossy: lossy,
		ch:    make(chan interface{}, size),
	}
}

// deliver reports whether the event was accepted. It must never block.
func (s *subscriber) deliver(event interface{}) bool {
	select {
	case s.ch <- event:
		return true
	default:
		s.dropped.Add(1)
		return false
	}
}

// Subscribe returns a channel carrying every event of the given types, with a
// backlog deep enough that control-flow events are not realistically lost.
func Subscribe(eventTypes ...interface{}) <-chan interface{} {
	s := newSubscriber(label(eventTypes), false)

	bus.mu.Lock()
	defer bus.mu.Unlock()

	for _, eventType := range eventTypes {
		t := reflect.TypeOf(eventType)
		bus.subscribers[t] = append(bus.subscribers[t], s)
	}
	return s.ch
}

// SubscribeAll returns a channel carrying every event published on the bus.
// The firehose is lossy: a consumer that stops draining loses events rather
// than stalling the bus.
func SubscribeAll() chan interface{} {
	s := newSubscriber("all", true)

	bus.mu.Lock()
	defer bus.mu.Unlock()

	bus.all = append(bus.all, s)
	return s.ch
}

// Unsubscribe removes ch from the bus. It holds the write lock, so it cannot
// overlap a Publish; once it returns no publisher holds a reference to ch and
// the caller may safely close it.
func Unsubscribe(ch chan interface{}) {
	bus.mu.Lock()
	defer bus.mu.Unlock()

	for i := len(bus.all) - 1; i >= 0; i-- {
		if bus.all[i].ch == ch {
			bus.all = append(bus.all[:i], bus.all[i+1:]...)
		}
	}
	for t, subs := range bus.subscribers {
		for i := len(subs) - 1; i >= 0; i-- {
			if subs[i].ch == ch {
				subs = append(subs[:i], subs[i+1:]...)
			}
		}
		bus.subscribers[t] = subs
	}
}

func Publish(event interface{}) {
	if n := publish(event); n > 0 {
		warnDropped(event, n)
	}
}

func publish(event interface{}) int {
	t := reflect.TypeOf(event)
	bus.mu.RLock()
	defer bus.mu.RUnlock()

	drops := 0

	// Send to type-specific subscribers
	if subs, found := bus.subscribers[t]; found {
		for _, s := range subs {
			if !s.deliver(event) {
				drops++
			}
		}
	}

	// Send to all subscribers
	for _, s := range bus.all {
		if !s.deliver(event) {
			drops++
		}
	}

	return drops
}

// warnDropped runs outside the read lock and at most once a second: a wedged
// subscriber drops events by the thousand, and writing that many log lines
// would just move the stall from the bus to stderr.
func warnDropped(event interface{}, n int) {
	total := dropped.Add(uint64(n))

	now := time.Now().UnixNano()
	last := lastWarn.Load()
	if now-last < int64(time.Second) || !lastWarn.CompareAndSwap(last, now) {
		return
	}

	slog.Warn("bus: subscriber behind, dropping events",
		"type", reflect.TypeOf(event),
		"subscribers", n,
		"total", total,
	)
}

// Stat reports one subscriber's backlog and losses.
type Stat struct {
	Label   string
	Lossy   bool
	Queued  int
	Dropped uint64
}

// Stats reports per-subscriber delivery health, so a hung consumer can be
// identified rather than merely detected.
func Stats() []Stat {
	bus.mu.RLock()
	seen := make(map[*subscriber]bool)
	subs := make([]*subscriber, 0, len(bus.all))
	for _, s := range bus.all {
		if !seen[s] {
			seen[s] = true
			subs = append(subs, s)
		}
	}
	for _, list := range bus.subscribers {
		for _, s := range list {
			if !seen[s] {
				seen[s] = true
				subs = append(subs, s)
			}
		}
	}
	bus.mu.RUnlock()

	stats := make([]Stat, 0, len(subs))
	for _, s := range subs {
		stats = append(stats, Stat{
			Label:   s.label,
			Lossy:   s.lossy,
			Queued:  len(s.ch),
			Dropped: s.dropped.Load(),
		})
	}
	return stats
}

// Dropped reports how many deliveries have been dropped across all subscribers.
func Dropped() uint64 {
	return dropped.Load()
}

func label(eventTypes []interface{}) string {
	names := make([]string, 0, len(eventTypes))
	for _, eventType := range eventTypes {
		names = append(names, reflect.TypeOf(eventType).String())
	}
	return strings.Join(names, ",")
}
