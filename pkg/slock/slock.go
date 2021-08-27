// slock - a simple, yet effective way for locking
package slock

import "sync/atomic"

const (
	stopped = iota
	running
)

// ServiceLock implements a very simple yet effective way to prevent services
// from multipple starting or stopping not running
// This implementation uses CPU effective atomic vars
type ServiceLock struct {
	serviceRunning uint32
}

func (sl *ServiceLock) TryLock() bool {
	return atomic.CompareAndSwapUint32(&sl.serviceRunning, stopped, running)
}

func (sl *ServiceLock) TryUnlock() bool {
	return atomic.CompareAndSwapUint32(&sl.serviceRunning, running, stopped)
}

func (sl *ServiceLock) Running() bool {
	return atomic.LoadUint32(&sl.serviceRunning) == running
}
