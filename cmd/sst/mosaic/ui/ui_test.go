package ui

import (
	"sync"
	"testing"
)

// The tunnel proxy calls GetColor(addr).Bold(true) from one goroutine per
// connection. This used to crash with "concurrent map writes" because every
// returned style shared the rules map of the global Colors entry.
func TestGetColorConcurrentUse(t *testing.T) {
	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 1000; j++ {
				GetColor("10.0.0.1:5432").Bold(true).Render("| ")
			}
		}()
	}
	wg.Wait()
}

func TestGetColorDoesNotMutateColors(t *testing.T) {
	GetColor("10.0.0.1:5432").Bold(true)
	for i, c := range Colors {
		if c.GetBold() {
			t.Fatalf("Colors[%d] was mutated through GetColor", i)
		}
	}
}
