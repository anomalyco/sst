package bus_test

import (
	"sync"
	"testing"

	"github.com/sst/sst/v3/pkg/bus"
)

// Each benchmark owns an event type and registers its subscribers once, so a
// repeated -count run measures the same steady state. Subscribers cannot be
// removed once registered, so run one benchmark per process (-bench) to keep
// them from bleeding into each other.

type benchNone struct{ Value int }
type benchFire1 struct{ Value int }
type benchFire16 struct{ Value int }
type benchTyped1 struct{ Value int }
type benchTyped16 struct{ Value int }
type benchParallel struct{ Value int }

var (
	fire1Once    sync.Once
	fire16Once   sync.Once
	typed1Once   sync.Once
	typed16Once  sync.Once
	parallelOnce sync.Once
)

func drainAll(n int) {
	for i := 0; i < n; i++ {
		ch := bus.SubscribeAll()
		go func() {
			for range ch {
			}
		}()
	}
}

func drainTyped(n int, eventType interface{}) {
	for i := 0; i < n; i++ {
		ch := bus.Subscribe(eventType)
		go func() {
			for range ch {
			}
		}()
	}
}

// run reports ns/op plus how many deliveries were dropped per publish, so a
// fast number produced by discarding events is visible rather than flattering.
func run(b *testing.B, publish func(i int)) {
	before := bus.Dropped()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		publish(i)
	}
	b.StopTimer()
	b.ReportMetric(float64(bus.Dropped()-before)/float64(b.N), "drops/op")
}

// Floor cost: type lookup and lock acquisition with nobody listening.
func BenchmarkPublishNoSubscribers(b *testing.B) {
	run(b, func(i int) { bus.Publish(benchNone{Value: i}) })
}

func BenchmarkPublishFirehose1(b *testing.B) {
	fire1Once.Do(func() { drainAll(1) })
	run(b, func(i int) { bus.Publish(benchFire1{Value: i}) })
}

func BenchmarkPublishFirehose16(b *testing.B) {
	fire16Once.Do(func() { drainAll(16) })
	run(b, func(i int) { bus.Publish(benchFire16{Value: i}) })
}

func BenchmarkPublishTyped1(b *testing.B) {
	typed1Once.Do(func() { drainTyped(1, benchTyped1{}) })
	run(b, func(i int) { bus.Publish(benchTyped1{Value: i}) })
}

func BenchmarkPublishTyped16(b *testing.B) {
	typed16Once.Do(func() { drainTyped(16, benchTyped16{}) })
	run(b, func(i int) { bus.Publish(benchTyped16{Value: i}) })
}

// Concurrent publishers, which is what the read lock actually sees in sst dev.
func BenchmarkPublishParallel(b *testing.B) {
	parallelOnce.Do(func() { drainTyped(4, benchParallel{}) })
	before := bus.Dropped()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			bus.Publish(benchParallel{})
		}
	})
	b.StopTimer()
	b.ReportMetric(float64(bus.Dropped()-before)/float64(b.N), "drops/op")
}
