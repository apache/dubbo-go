package interfaces

import "bytes"

// ConfigReader
type ConfigReader interface {
	ReadConsumerConfig(reader *bytes.Buffer) error
	ReadProviderConfig(reader *bytes.Buffer) error
}
