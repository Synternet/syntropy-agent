package logger

// nullWriter discards all messages
type nullWritter struct{}

func (w *nullWritter) Write(b []byte) (n int, err error) {
	return len(b), nil
}
