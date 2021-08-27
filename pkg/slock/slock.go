// slock - a simple, yet effective way for locking
package slock

import "sync/atomic"

const (
	stopped = iota
	running
)

// ServiceLocker is minimalistic service locking interface
type ServiceLocker interface {
	TryLock() bool
	TryUnlock() bool
	Running() bool
}

// ServiceLock implements a very simple yet effective way to prevent services
// from multipple starting or stopping not running
// This implementation uses CPU effective atomic vars
type AtomicServiceLock struct {
	serviceRunning uint32
}

func (sl *AtomicServiceLock) TryLock() bool {
	return atomic.CompareAndSwapUint32(&sl.serviceRunning, stopped, running)
}

func (sl *AtomicServiceLock) TryUnlock() bool {
	return atomic.CompareAndSwapUint32(&sl.serviceRunning, running, stopped)
}

func (sl *AtomicServiceLock) Running() bool {
	return atomic.LoadUint32(&sl.serviceRunning) == running
}
