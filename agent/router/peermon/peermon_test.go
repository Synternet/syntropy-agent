package peermon

import (
	"reflect"
	"testing"

	"github.com/SyntropyNet/syntropy-agent-go/pkg/multiping"
)

func TestPeerInfo_String(t *testing.T) {
	tests := []struct {
		name     string
		receiver peerInfo

		want1 string
	}{
		{
			"new",
			peerInfo{},
			" via  loss: 0.000000 latency 0.000000",
		},
		{
			"filled",
			peerInfo{endpoint: "abc", gateway: "cdd", latency: 1.23, loss: 3.14},
			"abc via cdd loss: 3.140000 latency 1.230000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			receiver := tt.receiver
			got1 := receiver.String()

			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("peerInfo.String got1 = %v, want1: %v", got1, tt.want1)
			}
		})
	}
}

func TestPeerMonitor_AddNode(t *testing.T) {
	type args struct {
		gw   string
		peer string
	}
	tests := []struct {
		name    string
		init    func(t *testing.T) *PeerMonitor
		inspect func(r *PeerMonitor, t *testing.T) //inspects receiver after test run

		args func(t *testing.T) args
	}{
		{
			"new",
			func(t *testing.T) *PeerMonitor {
				return &PeerMonitor{}
			},
			func(r *PeerMonitor, t *testing.T) {
				if len(r.peerList) != 1 {
					t.Fatal("Peer list was not updated")
				}
				if r.peerList[0].endpoint != "ep" || r.peerList[0].gateway != "gw" {
					t.Errorf("Peer entry not updated: got %v, want %v", r.peerList[0], peerInfo{endpoint: "ep", gateway: "gw"})
				}
			},
			func(t *testing.T) args {
				return args{"gw", "ep"}
			},
		},
		{
			"add",
			func(t *testing.T) *PeerMonitor {
				pm := &PeerMonitor{}
				pm.AddNode("abc", "def")
				return pm
			},
			func(r *PeerMonitor, t *testing.T) {
				if len(r.peerList) != 2 {
					t.Fatal("Peer list was not updated")
				}
				if r.peerList[1].endpoint != "ep" || r.peerList[1].gateway != "gw" {
					t.Errorf("Peer entry not updated: got %v, want %v", r.peerList[1], peerInfo{endpoint: "ep", gateway: "gw"})
				}
			},
			func(t *testing.T) args {
				return args{"gw", "ep"}
			},
		},
		{
			"duplicate ep",
			func(t *testing.T) *PeerMonitor {
				pm := &PeerMonitor{}
				pm.AddNode("gw", "ep")
				return pm
			},
			func(r *PeerMonitor, t *testing.T) {
				if len(r.peerList) != 1 {
					t.Fatal("Peer list was updated")
				}
			},
			func(t *testing.T) args {
				return args{"abc", "ep"}
			},
		},
		{
			"duplicate gw",
			func(t *testing.T) *PeerMonitor {
				pm := &PeerMonitor{}
				pm.AddNode("gw", "ep")
				return pm
			},
			func(r *PeerMonitor, t *testing.T) {
				if len(r.peerList) != 2 {
					t.Fatal("Peer list was not updated")
				}
			},
			func(t *testing.T) args {
				return args{"gw", "abc"}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tArgs := tt.args(t)

			receiver := tt.init(t)
			receiver.AddNode(tArgs.gw, tArgs.peer)

			if tt.inspect != nil {
				tt.inspect(receiver, t)
			}

		})
	}
}

func TestPeerMonitor_Peers(t *testing.T) {
	tests := []struct {
		name    string
		init    func(t *testing.T) *PeerMonitor
		inspect func(r *PeerMonitor, t *testing.T) //inspects receiver after test run

		want1 []string
	}{
		{
			"new",
			func(t *testing.T) *PeerMonitor {
				pm := &PeerMonitor{}
				pm.AddNode("gw", "ep")
				pm.AddNode("abc", "def")
				return pm
			},
			func(r *PeerMonitor, t *testing.T) {
				want := []*peerInfo{
					{
						endpoint: "ep",
						gateway:  "gw",
					},
					{
						endpoint: "def",
						gateway:  "abc",
					},
				}
				if !reflect.DeepEqual(r.peerList, want) {
					t.Errorf("PeerMonitor.Peers got = %v, want: %v", r.peerList, want)
				}
			},
			[]string{"ep", "def"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			receiver := tt.init(t)
			got1 := receiver.Peers()

			if tt.inspect != nil {
				tt.inspect(receiver, t)
			}

			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("PeerMonitor.Peers got1 = %v, want1: %v", got1, tt.want1)
			}
		})
	}
}

func defaultPeerMonitor(t *testing.T) *PeerMonitor {
	pm := &PeerMonitor{}
	pm.AddNode("1.1.1.1", "1.1.1.9")
	pm.AddNode("2.2.2.1", "2.2.2.9")
	pm.AddNode("3.3.3.1", "3.3.3.9")
	pm.AddNode("4.4.4.1", "4.4.4.9")
	return pm
}

func TestPeerMonitor_PingProcess(t *testing.T) {
	type args struct {
		pr []multiping.PingResult
	}
	tests := []struct {
		name    string
		init    func(t *testing.T) *PeerMonitor
		inspect func(r *PeerMonitor, t *testing.T) //inspects receiver after test run

		args func(t *testing.T) args
	}{
		{
			"out of order",
			defaultPeerMonitor,
			func(r *PeerMonitor, t *testing.T) {
				want := []*peerInfo{
					{gateway: "1.1.1.1", endpoint: "1.1.1.9", loss: 0, latency: 10},
					{gateway: "2.2.2.1", endpoint: "2.2.2.9", loss: 1, latency: 0},
					{gateway: "3.3.3.1", endpoint: "3.3.3.9", loss: 0, latency: 3},
					{gateway: "4.4.4.1", endpoint: "4.4.4.9", loss: 0, latency: 5},
				}
				if !reflect.DeepEqual(r.peerList, want) {
					t.Errorf("PeerMonitor.PingProcess got = %v, want: %v", r.peerList, want)
				}
			},
			func(t *testing.T) args {
				return args{
					[]multiping.PingResult{
						{IP: "3.3.3.9", Loss: 0, Latency: 3},
						{IP: "1.1.1.9", Loss: 0, Latency: 10},
						{IP: "4.4.4.9", Loss: 0, Latency: 5},
						{IP: "2.2.2.9", Loss: 1, Latency: 0},
					},
				}
			},
		},
		{
			"partial",
			defaultPeerMonitor,
			func(r *PeerMonitor, t *testing.T) {
				want := []*peerInfo{
					{gateway: "1.1.1.1", endpoint: "1.1.1.9", loss: 0.5, latency: 10},
					{gateway: "2.2.2.1", endpoint: "2.2.2.9", loss: 0, latency: 0},
					{gateway: "3.3.3.1", endpoint: "3.3.3.9", loss: 0, latency: 0},
					{gateway: "4.4.4.1", endpoint: "4.4.4.9", loss: 0, latency: 0},
				}
				if !reflect.DeepEqual(r.peerList, want) {
					t.Errorf("PeerMonitor.PingProcess got = %v, want: %v", r.peerList, want)
				}
			},
			func(t *testing.T) args {
				return args{
					[]multiping.PingResult{
						{IP: "1.1.1.9", Loss: 0.5, Latency: 10},
					},
				}
			},
		},
		{
			"unknown",
			defaultPeerMonitor,
			func(r *PeerMonitor, t *testing.T) {
				want := []*peerInfo{
					{gateway: "1.1.1.1", endpoint: "1.1.1.9", loss: 0, latency: 0},
					{gateway: "2.2.2.1", endpoint: "2.2.2.9", loss: 0, latency: 0},
					{gateway: "3.3.3.1", endpoint: "3.3.3.9", loss: 0, latency: 0},
					{gateway: "4.4.4.1", endpoint: "4.4.4.9", loss: 0, latency: 0},
				}
				if !reflect.DeepEqual(r.peerList, want) {
					t.Errorf("PeerMonitor.PingProcess got = %v, want: %v", r.peerList, want)
				}
			},
			func(t *testing.T) args {
				return args{
					[]multiping.PingResult{
						{IP: "1.2.3.4", Loss: 0.5, Latency: 10},
					},
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tArgs := tt.args(t)

			receiver := tt.init(t)
			receiver.PingProcess(tArgs.pr)

			if tt.inspect != nil {
				tt.inspect(receiver, t)
			}

		})
	}
}

func TestPeerMonitor_BestPath(t *testing.T) {
	tests := []struct {
		name string
		init func(t *testing.T) *PeerMonitor

		want1 string
	}{
		{
			"new",
			func(t *testing.T) *PeerMonitor {
				pm := defaultPeerMonitor(t)
				pm.PingProcess([]multiping.PingResult{
					{IP: "1.1.1.9", Loss: 0.2, Latency: 3}, // Medium result
					{IP: "2.2.2.9", Loss: 1, Latency: 0},   // Lowest Latency, but packet Loss
					{IP: "3.3.3.9", Loss: 0.1, Latency: 3}, // Expected best
					{IP: "4.4.4.9", Loss: 0.1, Latency: 5}, // Best is not the last
				})
				return pm
			},
			"3.3.3.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			receiver := tt.init(t)
			got1 := receiver.BestPath()

			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("PeerMonitor.BestPath got1 = %v, want1: %v", got1, tt.want1)
			}
		})
	}
}
