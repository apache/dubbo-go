package extension

import (
	"github.com/apache/dubbo-go/filter"
)

var (
	authenticators    = make(map[string]func() filter.Authenticator)
	accesskeyStorages = make(map[string]func() filter.AccessKeyStorage)
)

// SetAuthenticator put the fcn into map with name
func SetAuthenticator(name string, fcn func() filter.Authenticator) {
	authenticators[name] = fcn
}

// GetAuthenticator find the Authenticator with name
// if not found, it will panic
func GetAuthenticator(name string) filter.Authenticator {
	if authenticators[name] == nil {
		panic("authenticator for " + name + " is not existing, make sure you have import the package.")
	}
	return authenticators[name]()
}

// SetAccesskeyStorages will set the fcn into map with this name
func SetAccesskeyStorages(name string, fcn func() filter.AccessKeyStorage) {
	accesskeyStorages[name] = fcn
}

// GetAccesskeyStorages find the storage with the name.
// If not found, it will panic.
func GetAccesskeyStorages(name string) filter.AccessKeyStorage {
	if accesskeyStorages[name] == nil {
		panic("accesskeyStorages for " + name + " is not existing, make sure you have import the package.")
	}
	return accesskeyStorages[name]()
}
