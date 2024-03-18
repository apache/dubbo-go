package resources

import (
	"fmt"
	"strings"
)

const (
	Scheme       = "spiffe"
	URIPrefix    = Scheme + "://"
	URIPrefixLen = len(URIPrefix)
	// The default SPIFFE URL value for trust domain
	defaultTrustDomain    = "cluster.local"
	ServiceAccountSegment = "sa"
	NamespaceSegment      = "ns"
)

type Identity struct {
	TrustDomain    string
	Namespace      string
	ServiceAccount string
}

func ParseIdentity(s string) (Identity, error) {
	if !strings.HasPrefix(s, URIPrefix) {
		return Identity{}, fmt.Errorf("identity is not a spiffe format")
	}
	split := strings.Split(s[URIPrefixLen:], "/")
	if len(split) != 5 {
		return Identity{}, fmt.Errorf("identity is not a spiffe format")
	}
	if split[1] != NamespaceSegment || split[3] != ServiceAccountSegment {
		return Identity{}, fmt.Errorf("identity is not a spiffe format")
	}
	return Identity{
		TrustDomain:    split[0],
		Namespace:      split[2],
		ServiceAccount: split[4],
	}, nil
}

func (i Identity) String() string {
	return URIPrefix + i.TrustDomain + "/ns/" + i.Namespace + "/sa/" + i.ServiceAccount
}
