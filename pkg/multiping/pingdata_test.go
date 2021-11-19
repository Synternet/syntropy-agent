package multiping

import (
	"fmt"
	"testing"
)

func TestPingData(t *testing.T) {
	const maxCount = 222
	data := NewPingData()

	if data.Count() != 0 {
		t.Errorf("Invalid initial PingData count")
	}

	data.Add("127.0.0.1")
	data.Add("127.0.0.1")
	data.Add("127.0.0.1")
	if data.Count() != 1 {
		t.Errorf("Duplicate entries check failed")
	}

	for i := 2; i <= maxCount; i++ {
		data.Add(fmt.Sprintf("127.0.0.%d", i))
	}
	if data.Count() != maxCount {
		t.Errorf("Total count test failed")
	}

	data.Reset()
	if data.Count() != maxCount {
		t.Errorf("Data reset test failed")
	}
	data.Flush()
	if data.Count() != 0 {
		t.Errorf("Data flush test failed")
	}
}
