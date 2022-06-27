package xds

import (
	"fmt"
	"testing"
)

func TestBuildCertificate(t *testing.T) {
	fmt.Println("test begin ")
	certificate, err := buildCertificate()

	fmt.Println(err)
	fmt.Println(certificate)

}
