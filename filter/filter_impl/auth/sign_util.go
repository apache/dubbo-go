package auth

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/gob"
	"errors"
	"strings"
)

func Sign(metadata, key string) string {
	return doSign([]byte(metadata), key)
}

func SignWithParams(params []interface{}, metadata, key string) (string, error) {
	if params == nil || len(params) == 0 {
		return Sign(metadata, key), nil
	}

	data := append(params, metadata)
	if bytes, err := toBytes(data); err != nil {
		// TODO
		return "", errors.New("data convert to bytes failed")
	} else {
		return doSign(bytes, key), nil
	}
}

func toBytes(data []interface{}) ([]byte, error) {
	var b bytes.Buffer
	enc := gob.NewEncoder(&b)
	if err := enc.Encode(data); err != nil {
		return nil, errors.New("")
	}
	return b.Bytes(), nil
}

func doSign(bytes []byte, key string) string {
	sum256 := sha256.Sum256(bytes)
	return base64.URLEncoding.EncodeToString(sum256[:])
}

func IsEmpty(s string, allowSpace bool) bool {
	if len(s) == 0 {
		return true
	}
	if !allowSpace {
		if strings.TrimSpace(s) == "" {
			return true
		}
	}
	return false
}
