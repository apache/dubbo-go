package protocol

// Extension - Protocol
type Protocol interface {
	Export()
	Refer()
	Destroy()
}
