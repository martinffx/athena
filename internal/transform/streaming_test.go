package transform

// noopFlusher is a test helper that implements http.Flusher
type noopFlusher struct{}

func (f *noopFlusher) Flush() {}
