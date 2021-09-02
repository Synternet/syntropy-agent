package slock_test

import (
	"testing"

	"github.com/SyntropyNet/syntropy-agent-go/pkg/slock"
)

func TestLock(t *testing.T) {
	var sl slock.ServiceLocker
	sl = &slock.AtomicServiceLock{}

	if sl.Running() {
		t.Error("Expected to be unlocked")
	}

	if !sl.TryLock() {
		t.Error("Expected to be unlocked")
	}

	if sl.TryLock() {
		t.Error("Expected to be locked")
	}

	if !sl.Running() {
		t.Error("Expected to be locked")
	}

	if !sl.TryUnlock() {
		t.Error("Expected to be locked")
	}

	if sl.TryUnlock() {
		t.Error("Expected to be unlocked")
	}

	if sl.Running() {
		t.Error("Expected to be unlocked")
	}
}