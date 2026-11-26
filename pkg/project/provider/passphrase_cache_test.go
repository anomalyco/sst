package provider

import (
	"fmt"
	"sync"
	"testing"
)

// passphraseHome is a Home whose only working method is getPassphrase, so the
// test exercises the passphrase cache and nothing else.
type passphraseHome struct {
	Home
}

func (passphraseHome) getPassphrase(app, stage string) (string, error) {
	return "passphrase-for-" + app + stage, nil
}

// `sst secret list` reads the stage's secrets and the fallback secrets at the
// same time, so GetPassphrase is called from several goroutines at once. The
// cache must survive that: Go aborts the whole process on a concurrent map
// write ("fatal error: concurrent map writes"), and `go test -race` reports it.
func TestGetPassphraseIsSafeToCallConcurrently(t *testing.T) {
	var wg sync.WaitGroup
	for i := 0; i < 64; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			backend := passphraseHome{}
			stage := fmt.Sprintf("stage-%d", i%8)
			got, err := GetPassphrase(backend, "app", stage)
			if err != nil {
				t.Errorf("GetPassphrase: %v", err)
				return
			}
			if want := "passphrase-for-app" + stage; got != want {
				t.Errorf("GetPassphrase = %q, want %q", got, want)
			}
		}(i)
	}
	wg.Wait()
}
