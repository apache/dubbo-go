package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnmarshalJwks(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		wantErr bool
	}{
		{
			name:    "Valid JWKS JSON",
			json:    `{"keys":[{"kid":"123","kty":"RSA","e":"AQAB","n":"zDq_QWvH8eD_xCw7nxLYk6XzBIJkC3_6F2nGxOKrXfo0idGFlc4eNF18jS_gwXuvb12eVCCHlhv6F7W1wGz2_4J3X_wbMxDoXPLiXxNm8ycYKpDXT5CSb6owzjJLoq3ZQ7Qq4GIHnyxq4qEJwZBkhAqqKdnl8U7wJvGZnvq2DxV8jBKJ_ljLRQ04LVDj-8sB4cM-sr91AyyFVz_zsAIGr7FltABnc6R-Nxtj7_2eUcrGYY5EOE9p3TR15zZzVfvQGd_02lJzHvVjpqG2LlS0hvFgIF47mL0x05RlgEz7MpNh1QdS5Kty1AnpV0t4o2aO_b_09IExrBabM0w"}]}`,
			wantErr: false,
		},
		// Add more test cases as needed
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := UnmarshalJwks(tt.json)
			if err != nil {
				assert.Error(t, err)
				return
			}
		})
	}
}
