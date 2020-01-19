package rest_interface

type RestConfigReader interface {
	ReadConsumerConfig() *RestConsumerConfig
	ReadProviderConfig() *RestProviderConfig
}
