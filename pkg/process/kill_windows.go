//go:build windows
// +build windows

package process

import "os"

func sendTermSignal(process *os.Process) error {
	return process.Kill()
}

func sendKillSignal(process *os.Process) error {
	return process.Kill()
}
