package filter

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/protocol"
)

type AccessKeyPair struct {
	AccessKey    string `yaml:"accessKey"   json:"accessKey,omitempty" property:"accessKey"`
	SecretKey    string `yaml:"secretKey"   json:"secretKey,omitempty" property:"secretKey"`
	ConsumerSide string `yaml:"consumerSide"   json:"ConsumerSide,consumerSide" property:"consumerSide"`
	ProviderSide string `yaml:"providerSide"   json:"providerSide,omitempty" property:"providerSide"`
	Creator      string `yaml:"creator"   json:"creator,omitempty" property:"creator"`
	Options      string `yaml:"options"   json:"options,omitempty" property:"options"`
}

type Authenticator interface {
	Sign(protocol.Invocation, *common.URL) error
	Authenticate(protocol.Invocation, *common.URL) error
}

type AccesskeyStorage interface {
	GetAccesskeyPair(protocol.Invocation, *common.URL) *AccessKeyPair
}
