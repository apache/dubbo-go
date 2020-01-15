package extension

import (
	"github.com/apache/dubbo-go/filter"
)

var(
	authenticators    = make(map[string]func() filter.Authenticator)
	accesskeyStorages = make(map[string]func() filter.AccesskeyStorage)
)


func SetAuthenticator(name string, fcn func() filter.Authenticator) {
	authenticators[name] = fcn
}

func GetAuthenticator(name string) filter.Authenticator {
	if authenticators[name] == nil {
		panic("authenticator for " + name + " is not existing, make sure you have import the package.")
	}
	return authenticators[name]()
}

func SetAccesskeyStorages(name string, fcn func() filter.AccesskeyStorage) {
	accesskeyStorages[name] = fcn
}

func GetAccesskeyStorages(name string) filter.AccesskeyStorage {
	if accesskeyStorages[name] == nil {
		panic("accesskeyStorages for " + name + " is not existing, make sure you have import the package.")
	}
	return accesskeyStorages[name]()
}