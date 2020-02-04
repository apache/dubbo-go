package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
)

// get a signature string with given information, such as metadata or parameters
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
	if bytes, err := json.Marshal(data); err != nil {
		return nil, errors.New("")
	} else {
		return bytes, nil
	}
}

func doSign(bytes []byte, key string) string {
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write(bytes)
	signature := mac.Sum(nil)
	return base64.URLEncoding.EncodeToString(signature)
}

func IsEmpty(s string, allowSpace bool) bool {
	if len(s) == 0 {
		return true
	}
	if !allowSpace {
		return strings.TrimSpace(s) == ""
	}
	return false
}
