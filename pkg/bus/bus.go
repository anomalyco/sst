package bus

import (
	"reflect"
	"slices"
	"sync"
)

var (
	bus = &EventBus{
		subscribers: make(map[reflect.Type][]chan interface{}),
		all:         make([]chan interface{}, 0),
	}
)

type EventBus struct {
	subscribers map[reflect.Type][]chan interface{}
	mu          sync.RWMutex
	all         []chan interface{}
}

func Subscribe(eventTypes ...interface{}) <-chan interface{} {
	bus.mu.Lock()
	defer bus.mu.Unlock()

	ch := make(chan interface{}, 10_000)
	for _, eventType := range eventTypes {
		t := reflect.TypeOf(eventType)
		bus.subscribers[t] = append(bus.subscribers[t], ch)
	}
	return ch
}

func SubscribeAll() chan interface{} {
	bus.mu.Lock()
	defer bus.mu.Unlock()

	ch := make(chan interface{}, 10_000)
	bus.all = append(bus.all, ch)
	return ch
}

func Unsubscribe(ch chan interface{}) {
	bus.mu.Lock()
	defer bus.mu.Unlock()

	isTarget := func(c chan interface{}) bool { return c == ch }

	bus.all = slices.DeleteFunc(bus.all, isTarget)

	for t, channels := range bus.subscribers {
		bus.subscribers[t] = slices.DeleteFunc(channels, isTarget)
	}
}

func Publish(event interface{}) {
	t := reflect.TypeOf(event)
	bus.mu.RLock()
	defer bus.mu.RUnlock()

	// Send to type-specific subscribers
	if chans, found := bus.subscribers[t]; found {
		for _, ch := range chans {
			ch <- event
		}
	}

	// Send to all subscribers
	for _, ch := range bus.all {
		ch <- event
	}
}
