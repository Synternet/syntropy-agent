package multiping

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	ping "github.com/go-ping/ping"
	"github.com/golang/mock/gomock"
)

var (
	// This context is a sample context that should not be manipulated with
	// and used only for comparing objects
	dummyCtx = context.Background()
)

func TestNew(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	pingClientMock := NewMockPingClient(mockCtrl)

	type args struct {
		ctx context.Context
		p   PingClient
	}
	tests := []struct {
		name string
		args func(t *testing.T) args

		inspect func(r *MultiPing, t *testing.T)
	}{
		{
			"new",
			func(t *testing.T) args {
				return args{
					ctx: dummyCtx,
					p:   pingClientMock,
				}
			},
			func(r *MultiPing, t *testing.T) {
				if r.pingClient != pingClientMock {
					t.Error("Ping client was not set")
				}
				if r.ctx.Context() != dummyCtx {
					t.Error("Wrong context for newly created Multiping")
				}
				pinger, err := r.pinger("google.com")
				if err != nil {
					t.Errorf("Failed using pinger creator: %v", err)
				}
				p, ok := pinger.(*ping.Pinger)
				if ok {
					if p.Count != r.Count {
						t.Errorf("Failed to set count: got %v, expected %v", p.Count, r.Count)
					}
					if p.Timeout != r.Timeout {
						t.Errorf("Failed to set timeout: got %v, expected %v", p.Timeout, r.Timeout)
					}
					if !p.Privileged() {
						t.Error("Priviledged mode not set")
					}
				} else {
					t.Errorf("Improper type of created pinger: %v", p)
				}
				if r.hosts != nil {
					t.Errorf("Hosts slice expected to be empty: %v", r.hosts)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tArgs := tt.args(t)

			got1 := New(tArgs.ctx, tArgs.p)

			tt.inspect(got1, t)
		})
	}
}

func softEqualCheck(t *testing.T, got, need []string) {
	for _, expected := range need {
		exists := false
		for _, actual := range got {
			if expected == actual {
				exists = true
				break
			}
		}
		if !exists {
			t.Errorf("Failed to compare slices: got %v, expected %v", got, need)
			return
		}
	}
}

func TestMultiPing_AddHost(t *testing.T) {
	type args struct {
		hosts []string
	}
	tests := []struct {
		name    string
		init    func(t *testing.T) *MultiPing
		inspect func(r *MultiPing, t *testing.T) //inspects receiver after test run

		args func(t *testing.T) args
	}{
		{
			"new",
			func(t *testing.T) *MultiPing {
				return New(dummyCtx, nil)
			},
			func(r *MultiPing, t *testing.T) {
				softEqualCheck(t, r.hosts, []string{"a", "b"})
			},
			func(t *testing.T) args {
				return args{
					[]string{"a", "b"},
				}
			},
		},
		{
			"preadded",
			func(t *testing.T) *MultiPing {
				mp := New(dummyCtx, nil)
				mp.AddHost("a", "b", "c")
				return mp
			},
			func(r *MultiPing, t *testing.T) {
				softEqualCheck(t, r.hosts, []string{"a", "b", "c", "d"})
			},
			func(t *testing.T) args {
				return args{
					[]string{"d"},
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tArgs := tt.args(t)

			receiver := tt.init(t)
			receiver.AddHost(tArgs.hosts...)

			if tt.inspect != nil {
				tt.inspect(receiver, t)
			}

		})
	}
}

func TestMultiPing_DelHost(t *testing.T) {
	type args struct {
		hosts []string
	}
	tests := []struct {
		name    string
		init    func(t *testing.T) *MultiPing
		inspect func(r *MultiPing, t *testing.T) //inspects receiver after test run

		args func(t *testing.T) args
	}{
		{
			"single first",
			func(t *testing.T) *MultiPing {
				mp := New(dummyCtx, nil)
				mp.AddHost("a", "b", "c")
				return mp
			},
			func(r *MultiPing, t *testing.T) {
				softEqualCheck(t, r.hosts, []string{"b", "c"})
			},
			func(t *testing.T) args {
				return args{
					[]string{"a"},
				}
			},
		},
		{
			"single middle",
			func(t *testing.T) *MultiPing {
				mp := New(dummyCtx, nil)
				mp.AddHost("a", "b", "c")
				return mp
			},
			func(r *MultiPing, t *testing.T) {
				softEqualCheck(t, r.hosts, []string{"a", "c"})
			},
			func(t *testing.T) args {
				return args{
					[]string{"b"},
				}
			},
		},
		{
			"single last",
			func(t *testing.T) *MultiPing {
				mp := New(dummyCtx, nil)
				mp.AddHost("a", "b", "c")
				return mp
			},
			func(r *MultiPing, t *testing.T) {
				softEqualCheck(t, r.hosts, []string{"a", "b"})
			},
			func(t *testing.T) args {
				return args{
					[]string{"c"},
				}
			},
		},
		{
			"single does not exist",
			func(t *testing.T) *MultiPing {
				mp := New(dummyCtx, nil)
				mp.AddHost("a", "b", "c")
				return mp
			},
			func(r *MultiPing, t *testing.T) {
				softEqualCheck(t, r.hosts, []string{"a", "b", "c"})
			},
			func(t *testing.T) args {
				return args{
					[]string{"d"},
				}
			},
		},
		{
			"multiple",
			func(t *testing.T) *MultiPing {
				mp := New(dummyCtx, nil)
				mp.AddHost("a", "b", "c", "d", "e")
				return mp
			},
			func(r *MultiPing, t *testing.T) {
				softEqualCheck(t, r.hosts, []string{"b", "d"})
			},
			func(t *testing.T) args {
				return args{
					[]string{"a", "c", "e"},
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tArgs := tt.args(t)

			receiver := tt.init(t)
			receiver.DelHost(tArgs.hosts...)

			if tt.inspect != nil {
				tt.inspect(receiver, t)
			}

		})
	}
}

func TestMultiPing_Flush(t *testing.T) {
	tests := []struct {
		name    string
		init    func(t *testing.T) *MultiPing
		inspect func(r *MultiPing, t *testing.T) //inspects receiver after test run

	}{
		{
			"multiple",
			func(t *testing.T) *MultiPing {
				mp := New(dummyCtx, nil)
				mp.AddHost("a", "b", "c", "d", "e")
				return mp
			},
			func(r *MultiPing, t *testing.T) {
				if len(r.hosts) != 0 {
					t.Errorf("Hosts must be empty: %v", r.hosts)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			receiver := tt.init(t)
			receiver.Flush()

			if tt.inspect != nil {
				tt.inspect(receiver, t)
			}

		})
	}
}

func makePingerCreator(ctrl *gomock.Controller, runErr error, packetLoss float64) func(addr string) (Pinger, error) {
	return func(addr string) (Pinger, error) {
		pingerMock := NewMockPinger(ctrl)
		pingerMock.EXPECT().Run().Return(runErr)
		if packetLoss == 0 {
			pingerMock.EXPECT().Statistics().Return(&ping.Statistics{
				Addr:       addr,
				PacketLoss: float64(rand.Intn(99)),
				AvgRtt:     time.Duration(rand.Int63()),
			})
		} else if packetLoss > 0 {
			pingerMock.EXPECT().Statistics().Return(&ping.Statistics{
				Addr:       addr,
				PacketLoss: packetLoss,
				AvgRtt:     time.Duration(rand.Int63()),
			})
		}
		return pingerMock, nil
	}
}

func checkWaitGroup(t *testing.T, wg *sync.WaitGroup, mustPanic bool) {
	defer func() {
		if r := recover(); r != nil {
			if !mustPanic {
				t.Errorf("Unexpected WaitGroup .Done() call made?: %v", r)
			}
		}
	}()
	wg.Done()
	if mustPanic {
		t.Error("WaitGroup must have been emptied by this point. Missing .Done() call?")
	}
}

func TestMultiPing_pingHost(t *testing.T) {
	pingResults := make([]PingResult, 2)
	mockCtrl := gomock.NewController(t)
	pingClientMock := NewMockPingClient(mockCtrl)

	type args struct {
		wgroup    *sync.WaitGroup
		hostIndex int
		results   []PingResult
	}
	tests := []struct {
		name    string
		init    func(t *testing.T) (*MultiPing, *sync.WaitGroup)
		inspect func(r *MultiPing, wg *sync.WaitGroup, t *testing.T) //inspects receiver after test run

		args func(t *testing.T, wg *sync.WaitGroup) args
	}{
		{
			"first",
			func(t *testing.T) (*MultiPing, *sync.WaitGroup) {
				mp := New(dummyCtx, pingClientMock)
				mp.AddHost("a", "b", "c", "d", "e")
				mp.pinger = makePingerCreator(mockCtrl, nil, 0)

				var wg sync.WaitGroup
				wg.Add(1)
				return mp, &wg
			},
			func(r *MultiPing, wg *sync.WaitGroup, t *testing.T) {
				checkWaitGroup(t, wg, true)
			},
			func(t *testing.T, wg *sync.WaitGroup) args {
				return args{
					wgroup:    wg,
					hostIndex: 0,
					results:   pingResults,
				}
			},
		},
		{
			"second",
			func(t *testing.T) (*MultiPing, *sync.WaitGroup) {
				mp := New(dummyCtx, nil)
				mp.AddHost("a", "b", "c", "d", "e")
				mp.pinger = makePingerCreator(mockCtrl, nil, 0)
				var wg sync.WaitGroup
				wg.Add(1)
				return mp, &wg
			},
			func(r *MultiPing, wg *sync.WaitGroup, t *testing.T) {
				checkWaitGroup(t, wg, true)
			},
			func(t *testing.T, wg *sync.WaitGroup) args {
				return args{
					wgroup:    wg,
					hostIndex: 1,
					results:   pingResults,
				}
			},
		},
		{
			"high packet loss",
			func(t *testing.T) (*MultiPing, *sync.WaitGroup) {
				mp := New(dummyCtx, nil)
				mp.AddHost("a", "b", "c", "d", "e")
				mp.pinger = makePingerCreator(mockCtrl, nil, 150)
				var wg sync.WaitGroup
				wg.Add(1)
				return mp, &wg
			},
			func(r *MultiPing, wg *sync.WaitGroup, t *testing.T) {
				checkWaitGroup(t, wg, true)
			},
			func(t *testing.T, wg *sync.WaitGroup) args {
				return args{
					wgroup:    wg,
					hostIndex: 1,
					results:   pingResults,
				}
			},
		},
		{
			"run fails",
			func(t *testing.T) (*MultiPing, *sync.WaitGroup) {
				mp := New(dummyCtx, nil)
				mp.AddHost("a", "b", "c", "d", "e")
				mp.pinger = makePingerCreator(mockCtrl, fmt.Errorf("failed for whatever reason"), -1)
				var wg sync.WaitGroup
				wg.Add(1)
				return mp, &wg
			},
			func(r *MultiPing, wg *sync.WaitGroup, t *testing.T) {
				checkWaitGroup(t, wg, true)
			},
			func(t *testing.T, wg *sync.WaitGroup) args {
				return args{
					wgroup:    wg,
					hostIndex: 1,
					results:   pingResults,
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			receiver, wg := tt.init(t)

			tArgs := tt.args(t, wg)
			receiver.pingHost(tArgs.wgroup, tArgs.hostIndex, tArgs.results)

			if tt.inspect != nil {
				tt.inspect(receiver, wg, t)
			}
		})
	}
}

func timedAsyncCall(t *testing.T, f func(), ctx context.Context, timeout time.Duration) {
	callCtx, callDone := context.WithCancel(context.Background())
	wd, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var start time.Time
	go func() {
		defer func() {
			if err := recover(); err != nil {
				t.Errorf("Method panicked: %v", err)
			}
		}()
		start = time.Now()
		f()
		callDone()
	}()

	select {
	case <-wd.Done():
		t.Errorf("Method failed to return in %v", time.Since(start))
		return
	case <-ctx.Done():
		t.Error("Method was cancelled before it returned")
		return
	case <-callCtx.Done():
	}

	select {
	case <-wd.Done():
		t.Errorf("Method failed to return in %v", time.Since(start))
		return
	case <-ctx.Done():
		t.Logf("Method completed in %v", time.Since(start))
	}
}

func TestMultiPing_Ping(t *testing.T) {
	allHosts := []string{"a", "b", "c", "d", "e"}

	for i := range allHosts {
		resCtx, resultsReady := context.WithCancel(context.Background())
		defer resultsReady()
		var pingResults []PingResult
		hosts := allHosts
		rand.Shuffle(len(hosts), func(i, j int) {
			hosts[i], hosts[j] = hosts[j], hosts[i]
		})
		hosts = hosts[:i]

		mockCtrl := gomock.NewController(t)
		pingClientMock := NewMockPingClient(mockCtrl)
		pingClientMock.EXPECT().PingProcess(gomock.Any()).MaxTimes(1).Do(func(pr []PingResult) {
			pingResults = pr
			resultsReady()
		})

		pinger := New(dummyCtx, pingClientMock)
		pinger.pinger = func(s string) (Pinger, error) {
			p := NewMockPinger(mockCtrl)
			if len(hosts) > 0 {
				p.EXPECT().Run()
				p.EXPECT().Statistics().Return(&ping.Statistics{
					Addr:       s,
					PacketLoss: rand.Float64() * 100,
					AvgRtt:     time.Duration(rand.Int63()),
				})
			}
			return p, nil
		}

		pinger.AddHost(hosts...)

		t.Run(fmt.Sprintf("Ping test for: %v", hosts), func(tt *testing.T) {
			timedAsyncCall(tt, pinger.Ping, resCtx, time.Millisecond)

			rHosts := make([]string, len(pingResults))
			for i := range pingResults {
				rHosts[i] = pingResults[i].IP
			}
			softEqualCheck(tt, rHosts, hosts)
		})
	}
}

func TestMultiPing_StartStop(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	callsExpected := 1

	mockCtrl := gomock.NewController(t)
	pingClientMock := NewMockPingClient(mockCtrl)
	pingClientMock.EXPECT().PingProcess(gomock.Any()).MinTimes(callsExpected)

	pinger := New(ctx, pingClientMock)

	pinger.pinger = func(s string) (Pinger, error) {
		p := NewMockPinger(mockCtrl)
		p.EXPECT().Run()
		p.EXPECT().Statistics().Return(&ping.Statistics{
			Addr:       s,
			PacketLoss: rand.Float64() * 100,
			AvgRtt:     time.Duration(rand.Int63()),
		})
		return p, nil
	}

	pinger.AddHost("a", "b", "c")
	pinger.Period = time.Microsecond * 100

	pinger.Start()
	time.Sleep(time.Millisecond * 3)
	pinger.Stop()
	time.Sleep(time.Millisecond)
}

func TestMultiPing_StartCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var (
		callsMade     int64
		callsExpected int = 3
	)

	mockCtrl := gomock.NewController(t)
	pingClientMock := NewMockPingClient(mockCtrl)
	pingClientMock.EXPECT().PingProcess(gomock.Any()).MinTimes(callsExpected).MaxTimes(callsExpected).Do(func([]PingResult) {
		calls := atomic.AddInt64(&callsMade, 1)
		if calls >= int64(callsExpected) {
			cancel()
		}
	})

	pinger := New(ctx, pingClientMock)

	pinger.pinger = func(s string) (Pinger, error) {
		p := NewMockPinger(mockCtrl)
		p.EXPECT().Run()
		p.EXPECT().Statistics().Return(&ping.Statistics{
			Addr:       s,
			PacketLoss: rand.Float64() * 100,
			AvgRtt:     time.Duration(rand.Int63()),
		})
		return p, nil
	}

	pinger.AddHost("a", "b", "c")
	pinger.Period = time.Millisecond

	timedAsyncCall(t, pinger.Start, ctx, time.Millisecond*10)
	time.Sleep(time.Millisecond * 10)
}
