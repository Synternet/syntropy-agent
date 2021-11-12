// A simple wrapper to have State changes based on atomic variable
package state

import "sync/atomic"

type StateMachine struct {
	state uint32
}

func (stm *StateMachine) SetState(newState uint32) {
	atomic.StoreUint32(&stm.state, newState)
}

func (stm *StateMachine) GetState() uint32 {
	return atomic.LoadUint32(&stm.state)
}

func (stm *StateMachine) ChangeState(oldState, newState uint32) bool {
	return atomic.CompareAndSwapUint32(&stm.state, oldState, newState)
}
