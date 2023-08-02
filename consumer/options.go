package consumer

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/config/instance"
	"dubbo.apache.org/dubbo-go/v3/registry"
	"github.com/creasty/defaults"
	"github.com/dubbogo/gost/log/logger"
	"github.com/go-playground/validator/v10"
	perrors "github.com/pkg/errors"
	"net/url"
	"strconv"
	"strings"
)

type Options struct {
	// todo: confused
	RequestTimeout string `default:"3s" yaml:"request-timeout" json:"request-timeout,omitempty" property:"request-timeout"`
	ProxyFactory   string `default:"default" yaml:"proxy" json:"proxy,omitempty" property:"proxy"`
	// would influence reference
	AdaptiveService bool `default:"false" yaml:"adaptive-service" json:"adaptive-service" property:"adaptive-service"`
	// would influence hystrix
	FilterConf                     interface{} `yaml:"filter-conf" json:"filter-conf,omitempty" property:"filter-conf"`
	MaxWaitTimeForServiceDiscovery string      `default:"3s" yaml:"max-wait-time-for-service-discovery" json:"max-wait-time-for-service-discovery,omitempty" property:"max-wait-time-for-service-discovery"`
	// would influence reference
	MeshEnabled bool `yaml:"mesh-enabled" json:"mesh-enabled,omitempty" property:"mesh-enabled"`
	// todo: use idl to provide Interface
	//// unique
	//InterfaceName string `yaml:"interface"  json:"interface,omitempty" property:"interface"`
	// maybe use other way to replace this field?
	// would set default from consumer
	Check *bool `yaml:"check"  json:"check,omitempty" property:"check"`
	// todo: think about a more precise concept to represent URL
	// unique
	URL string `yaml:"url"  json:"url,omitempty" property:"url"`
	// would set default from consumer
	Filter string `yaml:"filter" json:"filter,omitempty" property:"filter"`
	// would set default from consumer
	Protocol string `yaml:"protocol"  json:"protocol,omitempty" property:"protocol"`
	// todo: figure out how to use this field
	// would set default from consumer
	RegistryIDs []string `yaml:"registry-ids"  json:"registry-ids,omitempty"  property:"registry-ids"`
	// would set default directly
	Cluster string `yaml:"cluster"  json:"cluster,omitempty" property:"cluster"`
	// unique, influenced by consumer
	Loadbalance string `yaml:"loadbalance"  json:"loadbalance,omitempty" property:"loadbalance"`
	// todo: move to runtime config
	// unique
	Retries string `yaml:"retries"  json:"retries,omitempty" property:"retries"`
	// would set default from root(Application)
	Group string `yaml:"group"  json:"group,omitempty" property:"group"`
	// would set default from root(Application)
	Version string `yaml:"version"  json:"version,omitempty" property:"version"`
	// unique
	Serialization string `yaml:"serialization" json:"serialization" property:"serialization"`
	// unique
	ProvidedBy string `yaml:"provided_by"  json:"provided_by,omitempty" property:"provided_by"`
	//// todo: move MethodConfig to consumeOptions
	//Methods []string `yaml:"methods"  json:"methods,omitempty" property:"methods"`
	// unique
	Async bool `yaml:"async"  json:"async,omitempty" property:"async"`
	// unique
	Params map[string]string `yaml:"params"  json:"params,omitempty" property:"params"`
	// unique
	Generic string `yaml:"generic"  json:"generic,omitempty" property:"generic"`
	// unique
	Sticky bool `yaml:"sticky"   json:"sticky,omitempty" property:"sticky"`
	//// unique
	//RequestTimeout string `yaml:"timeout"  json:"timeout,omitempty" property:"timeout"`
	// unique
	ForceTag bool `yaml:"force.tag"  json:"force.tag,omitempty" property:"force.tag"`
	// would set default from Consumer
	TracingKey       string `yaml:"tracing-key" json:"tracing-key,omitempty" propertiy:"tracing-key"`
	metaDataType     string
	MeshProviderPort int `yaml:"mesh-provider-port" json:"mesh-provider-port,omitempty" propertiy:"mesh-provider-port"`
	Registries       map[string]*RegistryOptions
}

type Option func(*Options)

func WithURL(url string) Option {
	return func(opts *Options) {
		opts.URL = url
	}
}

type ConsumeOptions struct {
	RequestTimeout string
	Retries        string
}

type ConsumeOption func(*ConsumeOptions)

func newDefaultConsumeOptions() *ConsumeOptions {
	return &ConsumeOptions{
		RequestTimeout: "",
		Retries:        "",
	}
}

func WithRequestTimeout(timeout string) ConsumeOption {
	return func(opts *ConsumeOptions) {
		opts.RequestTimeout = timeout
	}
}

func WithRetries(retries string) ConsumeOption {
	return func(opts *ConsumeOptions) {
		opts.Retries = retries
	}
}

type TracingOptions struct {
}

// RegistryOptions is the configuration of the registry center
type RegistryOptions struct {
	Protocol          string            `validate:"required" yaml:"protocol"  json:"protocol,omitempty" property:"protocol"`
	Timeout           string            `default:"5s" validate:"required" yaml:"timeout" json:"timeout,omitempty" property:"timeout"` // unit: second
	Group             string            `yaml:"group" json:"group,omitempty" property:"group"`
	Namespace         string            `yaml:"namespace" json:"namespace,omitempty" property:"namespace"`
	TTL               string            `default:"15m" yaml:"ttl" json:"ttl,omitempty" property:"ttl"` // unit: minute
	Address           string            `validate:"required" yaml:"address" json:"address,omitempty" property:"address"`
	Username          string            `yaml:"username" json:"username,omitempty" property:"username"`
	Password          string            `yaml:"password" json:"password,omitempty"  property:"password"`
	Simplified        bool              `yaml:"simplified" json:"simplified,omitempty"  property:"simplified"`
	Preferred         bool              `yaml:"preferred" json:"preferred,omitempty" property:"preferred"` // Always use this registry first if set to true, useful when subscribe to multiple registriesConfig
	Zone              string            `yaml:"zone" json:"zone,omitempty" property:"zone"`                // The region where the registry belongs, usually used to isolate traffics
	Weight            int64             `yaml:"weight" json:"weight,omitempty" property:"weight"`          // Affects traffic distribution among registriesConfig, useful when subscribe to multiple registriesConfig Take effect only when no preferred registry is specified.
	Params            map[string]string `yaml:"params" json:"params,omitempty" property:"params"`
	RegistryType      string            `yaml:"registry-type"`
	UseAsMetaReport   bool              `default:"true" yaml:"use-as-meta-report" json:"use-as-meta-report,omitempty" property:"use-as-meta-report"`
	UseAsConfigCenter bool              `default:"true" yaml:"use-as-config-center" json:"use-as-config-center,omitempty" property:"use-as-config-center"`
}

// Prefix dubbo.registries
func (RegistryOptions) Prefix() string {
	return constant.RegistryConfigPrefix
}

func (c *RegistryOptions) Init() error {
	if err := defaults.Set(c); err != nil {
		return err
	}
	return c.startRegistryOptions()
}

func (c *RegistryOptions) getUrlMap(roleType common.RoleType) url.Values {
	urlMap := url.Values{}
	urlMap.Set(constant.RegistryGroupKey, c.Group)
	urlMap.Set(constant.RegistryRoleKey, strconv.Itoa(int(roleType)))
	urlMap.Set(constant.RegistryKey, c.Protocol)
	urlMap.Set(constant.RegistryTimeoutKey, c.Timeout)
	// multi registry invoker weight label for load balance
	urlMap.Set(constant.RegistryKey+"."+constant.RegistryLabelKey, strconv.FormatBool(true))
	urlMap.Set(constant.RegistryKey+"."+constant.PreferredKey, strconv.FormatBool(c.Preferred))
	urlMap.Set(constant.RegistryKey+"."+constant.RegistryZoneKey, c.Zone)
	urlMap.Set(constant.RegistryKey+"."+constant.WeightKey, strconv.FormatInt(c.Weight, 10))
	urlMap.Set(constant.RegistryTTLKey, c.TTL)
	urlMap.Set(constant.ClientNameKey, clientNameID(c, c.Protocol, c.Address))

	for k, v := range c.Params {
		urlMap.Set(k, v)
	}
	return urlMap
}

func (c *RegistryOptions) startRegistryOptions() error {
	c.translateRegistryAddress()
	if c.UseAsMetaReport && isValid(c.Address) {
		if tmpUrl, err := c.toMetadataReportUrl(); err == nil {
			instance.SetMetadataReportInstanceByReg(tmpUrl)
		} else {
			return perrors.Wrap(err, "Start RegistryOptions failed.")
		}
	}
	return verify(c)
}

// toMetadataReportUrl translate the registry configuration to the metadata reporting url
func (c *RegistryOptions) toMetadataReportUrl() (*common.URL, error) {
	res, err := common.NewURL(c.Address,
		common.WithLocation(c.Address),
		common.WithProtocol(c.Protocol),
		common.WithUsername(c.Username),
		common.WithPassword(c.Password),
		common.WithParamsValue(constant.TimeoutKey, c.Timeout),
		common.WithParamsValue(constant.ClientNameKey, clientNameID(c, c.Protocol, c.Address)),
		common.WithParamsValue(constant.MetadataReportGroupKey, c.Group),
		common.WithParamsValue(constant.MetadataReportNamespaceKey, c.Namespace),
	)
	if err != nil || len(res.Protocol) == 0 {
		return nil, perrors.New("Invalid Registry Config.")
	}
	return res, nil
}

// translateRegistryAddress translate registry address
//
//	eg:address=nacos://127.0.0.1:8848 will return 127.0.0.1:8848 and protocol will set nacos
func (c *RegistryOptions) translateRegistryAddress() string {
	if strings.Contains(c.Address, "://") {
		u, err := url.Parse(c.Address)
		if err != nil {
			logger.Errorf("The registry url is invalid, error: %#v", err)
			panic(err)
		}
		c.Protocol = u.Scheme
		c.Address = strings.Join([]string{u.Host, u.Path}, "")
	}
	return c.Address
}

func (c *RegistryOptions) GetInstance(roleType common.RoleType) (registry.Registry, error) {
	u, err := c.toURL(roleType)
	if err != nil {
		return nil, err
	}
	// if the protocol == registry, set protocol the registry value in url.params
	if u.Protocol == constant.RegistryProtocol {
		u.Protocol = u.GetParam(constant.RegistryKey, "")
	}
	return extension.GetRegistry(u.Protocol, u)
}

func (c *RegistryOptions) toURL(roleType common.RoleType) (*common.URL, error) {
	address := c.translateRegistryAddress()
	var registryURLProtocol string
	if c.RegistryType == constant.RegistryTypeService {
		// service discovery protocol
		registryURLProtocol = constant.ServiceRegistryProtocol
	} else if c.RegistryType == constant.RegistryTypeInterface {
		registryURLProtocol = constant.RegistryProtocol
	} else {
		registryURLProtocol = constant.ServiceRegistryProtocol
	}
	return common.NewURL(registryURLProtocol+"://"+address,
		common.WithParams(c.getUrlMap(roleType)),
		common.WithParamsValue(constant.RegistrySimplifiedKey, strconv.FormatBool(c.Simplified)),
		common.WithParamsValue(constant.RegistryKey, c.Protocol),
		common.WithParamsValue(constant.RegistryNamespaceKey, c.Namespace),
		common.WithParamsValue(constant.RegistryTimeoutKey, c.Timeout),
		common.WithUsername(c.Username),
		common.WithPassword(c.Password),
		common.WithLocation(c.Address),
	)
}

func (c *RegistryOptions) toURLs(roleType common.RoleType) ([]*common.URL, error) {
	address := c.translateRegistryAddress()
	var urls []*common.URL
	var err error
	var registryURL *common.URL

	if !isValid(c.Address) {
		logger.Infof("Empty or N/A registry address found, the process will work with no registry enabled " +
			"which means that the address of this instance will not be registered and not able to be found by other consumer instances.")
		return urls, nil
	}

	if c.RegistryType == constant.RegistryTypeService {
		// service discovery protocol
		if registryURL, err = c.createNewURL(constant.ServiceRegistryProtocol, address, roleType); err == nil {
			urls = append(urls, registryURL)
		}
	} else if c.RegistryType == constant.RegistryTypeInterface {
		if registryURL, err = c.createNewURL(constant.RegistryProtocol, address, roleType); err == nil {
			urls = append(urls, registryURL)
		}
	} else if c.RegistryType == constant.RegistryTypeAll {
		if registryURL, err = c.createNewURL(constant.ServiceRegistryProtocol, address, roleType); err == nil {
			urls = append(urls, registryURL)
		}
		if registryURL, err = c.createNewURL(constant.RegistryProtocol, address, roleType); err == nil {
			urls = append(urls, registryURL)
		}
	} else {
		if registryURL, err = c.createNewURL(constant.ServiceRegistryProtocol, address, roleType); err == nil {
			urls = append(urls, registryURL)
		}
	}
	return urls, err
}

func (c *RegistryOptions) createNewURL(protocol string, address string, roleType common.RoleType) (*common.URL, error) {
	return common.NewURL(protocol+"://"+address,
		common.WithParams(c.getUrlMap(roleType)),
		common.WithParamsValue(constant.RegistrySimplifiedKey, strconv.FormatBool(c.Simplified)),
		common.WithParamsValue(constant.RegistryKey, c.Protocol),
		common.WithParamsValue(constant.RegistryNamespaceKey, c.Namespace),
		common.WithParamsValue(constant.RegistryTimeoutKey, c.Timeout),
		common.WithUsername(c.Username),
		common.WithPassword(c.Password),
		common.WithLocation(c.Address),
	)
}

const (
	defaultZKAddr          = "127.0.0.1:2181" // default registry address of zookeeper
	defaultNacosAddr       = "127.0.0.1:8848" // the default registry address of nacos
	defaultRegistryTimeout = "3s"             // the default registry timeout
)

type RegistryOptionsOpt func(config *RegistryOptions) *RegistryOptions

// NewRegistryOptionsWithProtocolDefaultPort New default registry config
// the input @protocol can only be:
// "zookeeper" with default addr "127.0.0.1:2181"
// "nacos" with default addr "127.0.0.1:8848"
func NewRegistryOptionsWithProtocolDefaultPort(protocol string) *RegistryOptions {
	switch protocol {
	case "zookeeper":
		return &RegistryOptions{
			Protocol: protocol,
			Address:  defaultZKAddr,
			Timeout:  defaultRegistryTimeout,
		}
	case "nacos":
		return &RegistryOptions{
			Protocol: protocol,
			Address:  defaultNacosAddr,
			Timeout:  defaultRegistryTimeout,
		}
	default:
		return &RegistryOptions{
			Protocol: protocol,
		}
	}
}

// NewRegistryOptions creates New RegistryOptions with @opts
func NewRegistryOptions(opts ...RegistryOptionsOpt) *RegistryOptions {
	newRegistryOptions := NewRegistryOptionsWithProtocolDefaultPort("")
	for _, v := range opts {
		newRegistryOptions = v(newRegistryOptions)
	}
	return newRegistryOptions
}

// WithRegistryProtocol returns RegistryOptionsOpt with given @regProtocol name
func WithRegistryProtocol(regProtocol string) RegistryOptionsOpt {
	return func(config *RegistryOptions) *RegistryOptions {
		config.Protocol = regProtocol
		return config
	}
}

// WithRegistryAddress returns RegistryOptionsOpt with given @addr registry address
func WithRegistryAddress(addr string) RegistryOptionsOpt {
	return func(config *RegistryOptions) *RegistryOptions {
		config.Address = addr
		return config
	}
}

// WithRegistryTimeOut returns RegistryOptionsOpt with given @timeout registry config
func WithRegistryTimeOut(timeout string) RegistryOptionsOpt {
	return func(config *RegistryOptions) *RegistryOptions {
		config.Timeout = timeout
		return config
	}
}

// WithRegistryGroup returns RegistryOptionsOpt with given @group registry group
func WithRegistryGroup(group string) RegistryOptionsOpt {
	return func(config *RegistryOptions) *RegistryOptions {
		config.Group = group
		return config
	}
}

// WithRegistryTTL returns RegistryOptionsOpt with given @ttl registry ttl
func WithRegistryTTL(ttl string) RegistryOptionsOpt {
	return func(config *RegistryOptions) *RegistryOptions {
		config.TTL = ttl
		return config
	}
}

// WithRegistryUserName returns RegistryOptionsOpt with given @userName registry userName
func WithRegistryUserName(userName string) RegistryOptionsOpt {
	return func(config *RegistryOptions) *RegistryOptions {
		config.Username = userName
		return config
	}
}

// WithRegistryPassword returns RegistryOptionsOpt with given @psw registry password
func WithRegistryPassword(psw string) RegistryOptionsOpt {
	return func(config *RegistryOptions) *RegistryOptions {
		config.Password = psw
		return config
	}
}

// WithRegistrySimplified returns RegistryOptionsOpt with given @simplified registry simplified flag
func WithRegistrySimplified(simplified bool) RegistryOptionsOpt {
	return func(config *RegistryOptions) *RegistryOptions {
		config.Simplified = simplified
		return config
	}
}

// WithRegistryPreferred returns RegistryOptions with given @preferred registry preferred flag
func WithRegistryPreferred(preferred bool) RegistryOptionsOpt {
	return func(config *RegistryOptions) *RegistryOptions {
		config.Preferred = preferred
		return config
	}
}

// WithRegistryWeight returns RegistryOptionsOpt with given @weight registry weight flag
func WithRegistryWeight(weight int64) RegistryOptionsOpt {
	return func(config *RegistryOptions) *RegistryOptions {
		config.Weight = weight
		return config
	}
}

// WithRegistryParams returns RegistryOptionsOpt with given registry @params
func WithRegistryParams(params map[string]string) RegistryOptionsOpt {
	return func(config *RegistryOptions) *RegistryOptions {
		config.Params = params
		return config
	}
}

// DynamicUpdateProperties update registry
func (c *RegistryOptions) DynamicUpdateProperties(updateRegistryOptions *RegistryOptions) {
	// if nacos's registry timeout not equal local root config's registry timeout , update.
	if updateRegistryOptions != nil && updateRegistryOptions.Timeout != c.Timeout {
		c.Timeout = updateRegistryOptions.Timeout
		logger.Infof("RegistryOptionss Timeout was dynamically updated, new value:%v", c.Timeout)
	}
}

// clientNameID unique identifier id for client
func clientNameID(config extension.Config, protocol, address string) string {
	return strings.Join([]string{config.Prefix(), protocol, address}, "-")
}

func isValid(addr string) bool {
	return addr != "" && addr != constant.NotAvailable
}

var validate *validator.Validate

func init() {
	validate = validator.New()
}

func verify(s interface{}) error {
	if err := validate.Struct(s); err != nil {
		errs := err.(validator.ValidationErrors)
		var slice []string
		for _, msg := range errs {
			slice = append(slice, msg.Error())
		}
		return perrors.New(strings.Join(slice, ","))
	}
	return nil
}
