package multiping

import (
	"context"
	"time"

	"github.com/SyntropyNet/syntropy-agent/pkg/multiping/pingdata"
	"github.com/SyntropyNet/syntropy-agent/pkg/multiping/pinger"
)

// Ping is blocking function and runs for mp.Timeout time and pings all hosts in data
func (mp *MultiPing) Ping(data *pingdata.PingData) {
	if data.Count() == 0 {
		return
	}

	// Lock the pinger - its instance may be reused by several clients
	mp.Lock()
	defer mp.Unlock()

	err := mp.restart()
	if err != nil {
		return
	}

	// Some subfunctions in goroutines will need this pointer to store ping results
	mp.pingData = data

	mp.ctx, mp.cancel = context.WithTimeout(context.Background(), mp.Timeout)
	defer mp.cancel()

	// This goroutine depends on rxChan and no need to add it to workgroup
	// It will terminate on channel close
	go mp.batchProcessPacket()

	// 2 receiver goroutines: separate for IPv4 and IPv6
	if mp.conn4 != nil {
		mp.wg.Add(1)
		mp.conn4.SetReadDeadline(time.Now().Add(mp.Timeout))
		go mp.batchRecvICMP(pinger.ProtocolIpv4)
	}
	if mp.conn6 != nil {
		mp.wg.Add(1)
		mp.conn6.SetReadDeadline(time.Now().Add(mp.Timeout))
		go mp.batchRecvICMP(pinger.ProtocolIpv6)
	}

	// 2 Sender goroutine workers:
	// one prepares message and other actually sends it
	mp.wg.Add(1)
	go mp.batchSendIcmp()
	go mp.batchPrepareIcmp()

	// wait for timeout and close connections
	<-mp.ctx.Done()
	mp.closeConnection()

	// wait for all goroutines to terminate and cleanup
	mp.wg.Wait()
	mp.cleanup()
}
