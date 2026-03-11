package bus_test

import (
	"testing"

	"github.com/sst/sst/v3/pkg/bus"
	"github.com/stretchr/testify/assert"
)

type testEventA struct{ Value string }
type testEventB struct{ Value int }

func TestBus(t *testing.T) {
	t.Run("subscribe and publish", func(t *testing.T) {
		ch := bus.Subscribe(testEventA{})
		bus.Publish(testEventA{Value: "hello"})
		evt := <-ch
		assert.Equal(t, testEventA{Value: "hello"}, evt)
	})

	t.Run("wrong type not received", func(t *testing.T) {
		ch := bus.Subscribe(testEventB{})
		bus.Publish(testEventA{Value: "nope"})

		select {
		case <-ch:
			t.Fatal("should not receive wrong type")
		default:
		}
	})

	t.Run("subscribe all", func(t *testing.T) {
		ch := bus.SubscribeAll()
		defer bus.Unsubscribe(ch)

		bus.Publish(testEventA{Value: "a"})
		bus.Publish(testEventB{Value: 1})

		evt1 := <-ch
		evt2 := <-ch
		assert.Equal(t, testEventA{Value: "a"}, evt1)
		assert.Equal(t, testEventB{Value: 1}, evt2)
	})

	t.Run("unsubscribe stops receiving", func(t *testing.T) {
		ch := bus.SubscribeAll()
		bus.Unsubscribe(ch)
		bus.Publish(testEventA{Value: "after unsub"})

		select {
		case <-ch:
			t.Fatal("should not receive after unsubscribe")
		default:
		}
	})

	t.Run("multiple subscribers", func(t *testing.T) {
		ch1 := bus.Subscribe(testEventA{})
		ch2 := bus.Subscribe(testEventA{})

		bus.Publish(testEventA{Value: "multi"})

		evt1 := <-ch1
		evt2 := <-ch2
		assert.Equal(t, testEventA{Value: "multi"}, evt1)
		assert.Equal(t, testEventA{Value: "multi"}, evt2)
	})
}
