package credentials

import (
	"fmt"
	"testing"
)

func TestNewCertManager(t *testing.T) {
	manager, err := NewCertManager()
	fmt.Println(err)
	fmt.Println(manager.GetCertificate())
	fmt.Println(manager.GetRootCertificate())
	fmt.Println()
}
