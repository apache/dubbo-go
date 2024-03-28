package filter

import (
	"dubbo.apache.org/dubbo-go/v3/istio/resources"
	"encoding/json"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestJwtAuthnFilterEngine_Filter(t *testing.T) {

	jsonStr := `{
  "providers": {
    "origins-0": {
      "provider_name": "origins-0",
      "issuer": "https://dubbo.apache.org/",
      "audiences": [
        "dev",
        "test"
      ],
      "local_jwks": {},
      "from_headers": [
        {
          "name": "Authorization",
          "value_prefix": "Bearer "
        }
      ]
    }
  },
  "rules": [
    {
      "match": {
        "action": "prefix",
        "path": "/"
      },
      "requires": {
        "provider_names": [
          "origins-0"
        ],
        "allow_missing": true
      }
    }
  ]
}
`
	jwksStr := `{"keys":[{"alg":"RS256","e":"AQAB","kid":"key1","kty":"RSA","n":"tI4hr2gYjv1YJ98eVn5IGyCqH7ylXOdx-QSwb-ERxvweJDUMZcLx62dIELXwM8Jv1iOl3pDRBvk6Gl7CYSqNblsXvtnKwCUBAkv7IKGjlIjZSIBhO6J7n33M3T-BDf0YBWHpHAEuOXOwIn19w0jeGSTgTF6u3AuGXa6lZMqtxKbaWsA2c621LB8hB2s6aAOe01FEl3-e037lsuIgMKYQJUBGfoGy4OlIukVZz3M3BwbJiuuIUVl_JjVR0Nt-FuM8rXOdjWFOgn1XUz269dUv22dm3u1EJ3wpJ12rfUDtANrosk-2d32qURLwkb7DeALj8pTCayT0Qz44YjBEtROo1w"},{"alg":"none","k":"Ym9ndXM","kid":"key2","kty":"oct"}]}`
	authentication := &resources.JwtAuthentication{}
	err := json.Unmarshal([]byte(jsonStr), authentication)
	if err != nil {
		t.Errorf("Unmarshal authentication Err %v", err)
	}
	keySet, err := jwk.Parse([]byte(jwksStr))
	if err != nil {
		t.Errorf("Unmarshal authentication Err %v", err)
	}
	authentication.Providers["origins-0"].LocalJwks.Keys = keySet

	tests := []struct {
		name           string
		headers        map[string]string
		authentication *resources.JwtAuthentication
		want           *JwtAuthnResult
		wantErr        bool
	}{
		{
			name:           "no jwt token in header",
			headers:        map[string]string{},
			authentication: authentication,
			want: &JwtAuthnResult{
				JwtVerfiyStatus:  JwtVerfiyStatusOK,
				TokenExists:      false,
				TokenVerified:    false,
				JwtToken:         nil,
				FindHeaderName:   "",
				FindProviderName: "",
				FindToken:        "",
			},
			wantErr: false,
		},

		{
			name: "wrong format jwt token in header",
			headers: map[string]string{
				":x-path":       "/",
				"authorization": "Bearerr   eyJhbGciOiJSUzI1NiIsImtpZCI6ImtleTEiLCJ0eXAiOiJKV1QifQ.eyJhdWQiOlsiZGV2Il0sImV4cCI6MjAyNjg2OTMwNiwiaWF0IjoxNzExNTA5MzA2LCJpc3MiOiJodHRwczovL2R1YmJvLmFwYWNoZS5vcmcvIiwic3ViIjoic3BpZmZlOi8vY2x1c3Rlci5sb2NhbC9ucy9mb28vc2EvaHR0cGJpbiJ9.k-WO1bW8LVeEAungU-73PdYwqMQWkxYa9gpGtZPnsZtZFut6gSfwGwSvFe4AXzAQp0Un817W9iukNus0smYcgl9Dn-tRrfN6JVQIROZl709iwD0NeK8iHxzn9EMyDQ716bJUgu5eHnLUFSwj34kOnMfuoQT-9ZomP1QVZU4l_-LkmitG6Czf7RFrylSVqvSwR1WDa7NQT6ERGzsQweBbkaFp0UjYOARI9VuHvnZaEw9oYwGX8-UK9VLcFdthiPL39sAspx0sTMCb3wPd_pfGl1tjYrRSEtuwlIhjpDiyVqweeII2j-NRGKb4fuVPK1joZUF2PIYPSU1tRMsxlisBzg",
			},
			authentication: authentication,
			want: &JwtAuthnResult{
				JwtVerfiyStatus:  JwtVerfiyStatusFailed,
				TokenExists:      true,
				TokenVerified:    false,
				JwtToken:         nil,
				FindHeaderName:   "",
				FindProviderName: "origins-0",
				FindToken:        "",
			},
			wantErr: false,
		},
		{
			name: "jwt token in header",
			headers: map[string]string{
				":x-path":       "/",
				"authorization": "Bearer eyJhbGciOiJSUzI1NiIsImtpZCI6ImtleTEiLCJ0eXAiOiJKV1QifQ.eyJhdWQiOlsiZGV2Il0sImV4cCI6MjAyNjg2OTMwNiwiaWF0IjoxNzExNTA5MzA2LCJpc3MiOiJodHRwczovL2R1YmJvLmFwYWNoZS5vcmcvIiwic3ViIjoic3BpZmZlOi8vY2x1c3Rlci5sb2NhbC9ucy9mb28vc2EvaHR0cGJpbiJ9.k-WO1bW8LVeEAungU-73PdYwqMQWkxYa9gpGtZPnsZtZFut6gSfwGwSvFe4AXzAQp0Un817W9iukNus0smYcgl9Dn-tRrfN6JVQIROZl709iwD0NeK8iHxzn9EMyDQ716bJUgu5eHnLUFSwj34kOnMfuoQT-9ZomP1QVZU4l_-LkmitG6Czf7RFrylSVqvSwR1WDa7NQT6ERGzsQweBbkaFp0UjYOARI9VuHvnZaEw9oYwGX8-UK9VLcFdthiPL39sAspx0sTMCb3wPd_pfGl1tjYrRSEtuwlIhjpDiyVqweeII2j-NRGKb4fuVPK1joZUF2PIYPSU1tRMsxlisBzg",
			},
			authentication: authentication,
			want: &JwtAuthnResult{
				JwtVerfiyStatus:  JwtVerfiyStatusOK,
				TokenExists:      true,
				TokenVerified:    true,
				JwtToken:         nil,
				FindHeaderName:   "authorization",
				FindProviderName: "origins-0",
				FindToken:        "eyJhbGciOiJSUzI1NiIsImtpZCI6ImtleTEiLCJ0eXAiOiJKV1QifQ.eyJhdWQiOlsiZGV2Il0sImV4cCI6MjAyNjg2OTMwNiwiaWF0IjoxNzExNTA5MzA2LCJpc3MiOiJodHRwczovL2R1YmJvLmFwYWNoZS5vcmcvIiwic3ViIjoic3BpZmZlOi8vY2x1c3Rlci5sb2NhbC9ucy9mb28vc2EvaHR0cGJpbiJ9.k-WO1bW8LVeEAungU-73PdYwqMQWkxYa9gpGtZPnsZtZFut6gSfwGwSvFe4AXzAQp0Un817W9iukNus0smYcgl9Dn-tRrfN6JVQIROZl709iwD0NeK8iHxzn9EMyDQ716bJUgu5eHnLUFSwj34kOnMfuoQT-9ZomP1QVZU4l_-LkmitG6Czf7RFrylSVqvSwR1WDa7NQT6ERGzsQweBbkaFp0UjYOARI9VuHvnZaEw9oYwGX8-UK9VLcFdthiPL39sAspx0sTMCb3wPd_pfGl1tjYrRSEtuwlIhjpDiyVqweeII2j-NRGKb4fuVPK1joZUF2PIYPSU1tRMsxlisBzg",
			},
			wantErr: false,
		},
		{
			name: "error jwt token in header",
			headers: map[string]string{
				":x-path":       "/",
				"authorization": "Bearer wrongttoken",
			},
			authentication: authentication,
			want: &JwtAuthnResult{
				JwtVerfiyStatus:  JwtVerfiyStatusFailed,
				TokenExists:      true,
				TokenVerified:    false,
				JwtToken:         nil,
				FindHeaderName:   "authorization",
				FindProviderName: "origins-0",
				FindToken:        "wrongttoken",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &JwtAuthnFilterEngine{
				headers:        tt.headers,
				authentication: tt.authentication,
			}
			result, err := e.Filter()
			result.JwtToken = nil
			if (err != nil) != tt.wantErr {
				t.Errorf("Filter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, tt.want, result)
		})
	}
}
