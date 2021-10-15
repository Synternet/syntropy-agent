package router

import "testing"

func TestEntry(t *testing.T) {
	rl := routeList{}

	if rl.Count() != 0 {
		t.Error("Invalid route list initialise")
	}

	// Add few routes and test
	rl.Add(&routeEntry{
		ifname:  "eth0",
		gateway: "1.1.1.1",
	})
	rl.Add(&routeEntry{
		ifname:  "eth1",
		gateway: "2.2.2.2",
	})
	if rl.Count() != 2 {
		t.Error("Invalid route list cout")
	}

	// Add dupplicate entry
	rl.Add(&routeEntry{
		ifname:  "eth1",
		gateway: "2.2.2.2",
	})
	if rl.Count() != 2 {
		t.Error("Dupplicate route added")
	}

}
