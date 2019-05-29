package utils

import (
	"testing"
)
import (
	"github.com/stretchr/testify/assert"
)

func Test_RegSplit(t *testing.T) {
	strings := RegSplit("dubbo://123.1.2.1;jsonrpc://127.0.0.1;registry://3.2.1.3?registry=zookeeper", "\\s*[;]+\\s*")
	assert.Len(t, strings, 3)
	assert.Equal(t, "dubbo://123.1.2.1", strings[0])
	assert.Equal(t, "jsonrpc://127.0.0.1", strings[1])
	assert.Equal(t, "registry://3.2.1.3?registry=zookeeper", strings[2])
}
