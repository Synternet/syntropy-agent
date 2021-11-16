package webip

import (
	"testing"
)

func TestWebIP(t *testing.T) {
	// not much to test here.
	// it just should not return error and return valid address
	ip, err := PublicIP()

	if err != nil {
		t.Errorf("WebIP service failed %s", err)
	}
	if ip.IsUnspecified() {
		t.Errorf("Failed getting public IP.")
	}
}
