package resources

import (
	"github.com/golang/protobuf/ptypes/duration"
	"github.com/lestrrat-go/jwx/v2/jwk"
)

// JwtAuthnFilter definine jwt authn filter configuration
type JwtAuthnFilter struct {
	Name              string
	JwtAuthentication *JwtAuthentication
}

// JwtAuthentication defines the JWT authentication configuration for Envoy HTTP filters.
// Map of provider names to JwtProviders.
//
// .. code-block:: yaml
//
//	providers:
//	  provider1:
//	     issuer: issuer1
//	     audiences:
//	     - audience1
//	     - audience2
//	     remote_jwks:
//	       http_uri:
//	         uri: https://example.com/.well-known/jwks.json
//	         cluster: example_jwks_cluster
//	         timeout: 1s
//	   provider2:
//	     issuer: provider2
//	     local_jwks:
//	       inline_string: jwks_string
//
// Specifies requirements based on the route matches. The first matched requirement will be
// applied. If there are overlapped match conditions, please put the most specific match first.
//
// # Examples
//
// .. code-block:: yaml
//
//	rules:
//	  - match:
//	      prefix: /healthz
//	  - match:
//	      prefix: /baz
//	    requires:
//	      provider_name: provider1
//	  - match:
//	      prefix: /foo
//	    requires:
//	      requires_any:
//	        requirements:
//	          - provider_name: provider1
//	          - provider_name: provider2
//	  - match:
//	      prefix: /bar
//	    requires:
//	      requires_all:
//	        requirements:
//	          - provider_name: provider1
//	          - provider_name: provider2
type JwtAuthentication struct {
	Providers map[string]*JwtProvider `json:"providers,omitempty"`
	Rules     []*JwtRequirementRule   `json:"rules,omitempty"`
	//RequirementMap map[string]*JwtRequirement `json:"requirement_map,omitempty"`
}

// JwtProvider defines the JWT provider configuration.
type JwtProvider struct {
	ProviderName         string      `json:"provider_name,omitempty"`
	Issuer               string      `json:"issuer,omitempty"`
	Audiences            []string    `json:"audiences,omitempty"`
	LocalJwks            *LocalJwks  `json:"local_jwks,omitempty"`
	Forward              bool        `json:"forward,omitempty"`
	ForwardPayloadHeader string      `json:"forward_payload_header,omitempty"`
	FromHeaders          []JwtHeader `json:"from_headers,omitempty"`
}

// JwtHeader defines the header information used to extract JWT tokens from HTTP requests.
type JwtHeader struct {
	Name        string `json:"name,omitempty"`
	ValuePrefix string `json:"value_prefix,omitempty"`
}

// RemoteJwks defines the remote jwks
type RemoteJwks struct {
	// Not support now
}

// LocalJwks defines the local inline jwks
type LocalJwks struct {
	InlineString string
	Keys         jwk.Set
}

// JwksAsyncFetch defines the behavior of asynchronously fetching JWKS on the main thread.
type JwksAsyncFetch struct {
	FastListener          bool               `json:"fast_listener,omitempty"`
	FailedRefetchDuration *duration.Duration `json:"failed_refetch_duration,omitempty"`
}

// ProviderWithAudiences defines a JWT provider with audiences.
type ProviderWithAudiences struct {
	ProviderName string   `json:"provider_name,omitempty"`
	Audiences    []string `json:"audiences,omitempty"`
}

type JwtVerfiyStatus int32

const (
	JwtVerfiyStatusOK JwtVerfiyStatus = iota
	JwtVerfiyStatusMissing
	JwtVerfiyStatusFailed
)

type RequirementType uint32

const (
	RequirementTypeProviderName RequirementType = iota
	RequirementTypeProviderAndAudiences
	RequirementTypeAny
	RequirementTypeAll
)

// SimpleJwtRequirement 定义 requires 包含所有的 provider_name, 这里先忽略所有 OR 和 AND 逻辑,
// 这里采用全部 provider name 实行 OR 逻辑
type SimpleJwtRequirement struct {
	ProviderNames        []string `json:"provider_names,omitempty"`
	AllowMissingOrFailed bool     `json:"allow_missing_or_failed,omitempty"`
	AllowMissing         bool     `json:"allow_missing,omitempty"`
}

// JwtRequirement defines a JWT requirement.
type JwtRequirement struct {
	RequireType          RequirementType        `json:"require_type,omitempty"`
	ProviderName         string                 `json:"provider_name,omitempty"`
	ProviderAndAudiences *ProviderWithAudiences `json:"provider_and_audiences,omitempty"`
	RequiresAny          *JwtRequirementOrList  `json:"requires_any,omitempty"`
	RequiresAll          *JwtRequirementAndList `json:"requires_all,omitempty"`
	AllowMissingOrFailed bool                   `json:"allow_missing_or_failed,omitempty"`
	AllowMissing         bool                   `json:"allow_missing,omitempty"`
}

// JwtRequirementOrList defines a list of JWT requirements whose results are combined with OR logic.
type JwtRequirementOrList struct {
	Requirements []*JwtRequirement `json:"requirements,omitempty"`
}

// JwtRequirementAndList defines a list of JWT requirements whose results are combined with AND logic.
type JwtRequirementAndList struct {
	Requirements []*JwtRequirement `json:"requirements,omitempty"`
}

// JwtRequirementRule defines JWT requirement rules based on route matches.
type JwtRequirementRule struct {
	Match           *JwtRouteMatch        `json:"match,omitempty"`
	Requires        *SimpleJwtRequirement `json:"requires,omitempty"`
	RequirementName string                `json:"requirement_name,omitempty"`
}

// JwtRouteMatch defines route matching conditions.
type JwtRouteMatch struct {
	Action        string `json:"action,omitempty"`
	Value         string `json:"path,omitempty"`
	CaseSensitive bool   `json:"case_sensitive,omitempty"`
}

// UnmarshalJwks unmarshals JWKS JSON into []jwk.Set.
// Demo url:  https://www.googleapis.com/oauth2/v3/certs
func UnmarshalJwks(jwksJSON string) (jwk.Set, error) {
	var jwks jwk.Set
	jwks, err := jwk.Parse([]byte(jwksJSON))
	return jwks, err
}
