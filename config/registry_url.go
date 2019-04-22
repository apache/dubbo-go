package config

import "fmt"

/////////////////////////////////
// dubbo role type
/////////////////////////////////

const (
	CONSUMER = iota
	CONFIGURATOR
	ROUTER
	PROVIDER
)

var (
	DubboNodes = [...]string{"consumers", "configurators", "routers", "providers"}
	DubboRole  = [...]string{"consumer", "", "", "provider"}
)

type DubboType int

func (t DubboType) String() string {
	return DubboNodes[t]
}

func (t DubboType) Role() string {
	return DubboRole[t]
}

type RegistryURL struct {
	baseUrl
	//store for service waiting for register
	URL URL
	//both for registry & service & reference
	Version string `yaml:"version" json:"version,omitempty"`
	Group   string `yaml:"group" json:"group,omitempty"`
	//for registry
	Username     string    `yaml:"username" json:"username,omitempty"`
	Password     string    `yaml:"password" json:"password,omitempty"`
	DubboType    DubboType `yaml:"-"`
	Organization string    `yaml:"organization" json:"organization,omitempty"`
	Name         string    `yaml:"name" json:"name,omitempty"`
	Module       string    `yaml:"module" json:"module,omitempty"`
	Owner        string    `yaml:"owner" json:"owner,omitempty"`
	Environment  string    `yaml:"environment" json:"environment,omitempty"`
	Address      string    `yaml:"address" json:"address,omitempty"`
}

func (c *RegistryURL) Key() string {
	return fmt.Sprintf("%s-%s-%s-%s-%s",  c.Protocol, c.Group, c.Location, c.Version, c.DubboType)
}

func (c *RegistryURL) URLEqual(url IURL) bool {

	if c.Key() != url.Key() {
		return false
	}
	return true
}
