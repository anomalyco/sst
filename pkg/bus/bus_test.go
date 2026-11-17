package bus_test

import (
	"sync"
	"testing"
	"time"

	"github.com/sst/sst/v3/pkg/bus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testEventA struct{ Value string }
type testEventB struct{ Value int }
type testEventC struct{ Value int }
type testEventD struct{ Value int }
type testEventE struct{ Value string }
type testEventF struct{ Value int }
type testEventG struct{ Value int }

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
		case <-time.After(100 * time.Millisecond):
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

// A typed subscriber carries control-flow events, so a burst well past what
// the firehose could hold must survive undropped and in order.
func TestTypedSubscriberIsLossless(t *testing.T) {
	const count = 12_000

	ch := bus.Subscribe(testEventC{})
	before := bus.Dropped()

	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < count; i++ {
			bus.Publish(testEventC{Value: i})
		}
	}()

	select {
	case <-done:
	case <-time.After(30 * time.Second):
		t.Fatal("Publish blocked on a typed subscriber")
	}

	for i := 0; i < count; i++ {
		select {
		case evt := <-ch:
			require.Equal(t, testEventC{Value: i}, evt, "event %d out of order", i)
		case <-time.After(30 * time.Second):
			t.Fatalf("only received %d of %d events", i, count)
		}
	}

	assert.Equal(t, before, bus.Dropped(), "typed subscriber should not drop")
}

// The firehose is lossy on purpose: a consumer that stops reading must not
// stall publishers or the bus lock.
func TestFirehoseDropsInsteadOfBlocking(t *testing.T) {
	stuck := bus.SubscribeAll()
	defer func() {
		bus.Unsubscribe(stuck)
		close(stuck)
	}()

	before := bus.Dropped()

	done := make(chan struct{})
	go func() {
		defer close(done)
		// One more than the channel buffer, so the last send has nowhere to go.
		for i := 0; i < 10_001; i++ {
			bus.Publish(testEventD{Value: i})
		}
	}()

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("Publish blocked on a full firehose subscriber")
	}

	assert.Greater(t, bus.Dropped(), before, "expected overflow to be counted as dropped")

	// Subscribe takes the write lock; it hangs forever if a publisher is still
	// parked on the read lock trying to send to the full subscriber above.
	subscribed := make(chan struct{})
	go func() {
		defer close(subscribed)
		bus.Subscribe(testEventE{})
	}()

	select {
	case <-subscribed:
	case <-time.After(10 * time.Second):
		t.Fatal("Subscribe blocked while a subscriber was full")
	}
}

func TestStatsIdentifyTheWedgedSubscriber(t *testing.T) {
	stuck := bus.SubscribeAll()
	defer func() {
		bus.Unsubscribe(stuck)
		close(stuck)
	}()

	for i := 0; i < 10_001; i++ {
		bus.Publish(testEventD{Value: i})
	}

	var lossy *bus.Stat
	for _, stat := range bus.Stats() {
		if stat.Lossy && stat.Dropped > 0 {
			lossy = &stat
			break
		}
	}
	require.NotNil(t, lossy, "expected a lossy subscriber with drops in Stats")
	assert.Equal(t, "all", lossy.Label)
}

// Unsubscribe must fence off every sender before the caller closes the channel,
// which is the pattern used by deploy, remove, refresh and diff.
func TestUnsubscribeThenCloseIsSafe(t *testing.T) {
	var wg sync.WaitGroup
	stop := make(chan struct{})

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
				bus.Publish(testEventE{Value: "concurrent"})
			}
		}
	}()

	for i := 0; i < 50; i++ {
		ch := bus.SubscribeAll()
		go func() {
			for range ch {
			}
		}()
		time.Sleep(time.Millisecond)
		bus.Unsubscribe(ch)
		close(ch)
	}

	close(stop)
	wg.Wait()
}

// The deeper buffer is not a blocking one: a typed subscriber that stops
// draining must still not stall publishers or wedge the bus lock.
func TestFullTypedSubscriberDoesNotFreezeBus(t *testing.T) {
	bus.Subscribe(testEventF{})

	before := bus.Dropped()

	done := make(chan struct{})
	go func() {
		defer close(done)
		// Comfortably past typedBuffer, with nobody reading.
		for i := 0; i < 20_000; i++ {
			bus.Publish(testEventF{Value: i})
		}
	}()

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("Publish blocked on a full typed subscriber")
	}

	assert.Greater(t, bus.Dropped(), before, "overflow past the buffer should drop, not block")

	// Subscribe takes the write lock; it hangs if a publisher is still parked
	// on the read lock trying to send to the full subscriber above.
	subscribed := make(chan struct{})
	go func() {
		defer close(subscribed)
		bus.Subscribe(testEventG{})
	}()

	select {
	case <-subscribed:
	case <-time.After(10 * time.Second):
		t.Fatal("Subscribe blocked while a typed subscriber was full")
	}
}
