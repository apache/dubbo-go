package client

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"fmt"
	"github.com/go-playground/validator/v10"
	perrors "github.com/pkg/errors"
	"regexp"
	"strconv"
	"strings"
)

type CallOptions struct {
	RequestTimeout string
	Retries        string
}

type CallOption func(*CallOptions)

func newDefaultCallOptions() *CallOptions {
	return &CallOptions{
		RequestTimeout: "",
		Retries:        "",
	}
}

func WithCallRequestTimeout(timeout string) CallOption {
	return func(opts *CallOptions) {
		opts.RequestTimeout = timeout
	}
}

func WithCallRetries(retries string) CallOption {
	return func(opts *CallOptions) {
		opts.Retries = retries
	}
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

func mergeValue(str1, str2, def string) string {
	if str1 == "" && str2 == "" {
		return def
	}
	s1 := strings.Split(str1, ",")
	s2 := strings.Split(str2, ",")
	str := "," + strings.Join(append(s1, s2...), ",")
	defKey := strings.Contains(str, ","+constant.DefaultKey)
	if !defKey {
		str = "," + constant.DefaultKey + str
	}
	str = strings.TrimPrefix(strings.Replace(str, ","+constant.DefaultKey, ","+def, -1), ",")
	return removeMinus(strings.Split(str, ","))
}

func removeMinus(strArr []string) string {
	if len(strArr) == 0 {
		return ""
	}
	var normalStr string
	var minusStrArr []string
	for _, v := range strArr {
		if strings.HasPrefix(v, "-") {
			minusStrArr = append(minusStrArr, v[1:])
		} else {
			normalStr += fmt.Sprintf(",%s", v)
		}
	}
	normalStr = strings.Trim(normalStr, ",")
	for _, v := range minusStrArr {
		normalStr = strings.Replace(normalStr, v, "", 1)
	}
	reg := regexp.MustCompile("[,]+")
	normalStr = reg.ReplaceAllString(strings.Trim(normalStr, ","), ",")
	return normalStr
}

// ----------ReferenceOption----------

func WithCheck(check bool) ReferenceOption {
	return func(cfg *ReferenceConfig) {
		cfg.Check = &check
	}
}

func WithURL(url string) ReferenceOption {
	return func(cfg *ReferenceConfig) {
		cfg.URL = url
	}
}

func WithFilter(filter string) ReferenceOption {
	return func(cfg *ReferenceConfig) {
		cfg.Filter = filter
	}
}

func WithProtocol(protocol string) ReferenceOption {
	return func(cfg *ReferenceConfig) {
		cfg.Protocol = protocol
	}
}

// todo: think about a more intuitive way
func WithRegistryIDs(registryIDs []string) ReferenceOption {
	return func(cfg *ReferenceConfig) {
		cfg.RegistryIDs = registryIDs
	}
}

func WithCluster(cluster string) ReferenceOption {
	return func(cfg *ReferenceConfig) {
		cfg.Cluster = cluster
	}
}

func WithLoadBalance(loadBalance string) ReferenceOption {
	return func(cfg *ReferenceConfig) {
		cfg.Loadbalance = loadBalance
	}
}

func WithRetries(retries int) ReferenceOption {
	return func(cfg *ReferenceConfig) {
		cfg.Retries = strconv.Itoa(retries)
	}
}

func WithGroup(group string) ReferenceOption {
	return func(cfg *ReferenceConfig) {
		cfg.Group = group
	}
}

func WithVersion(version string) ReferenceOption {
	return func(cfg *ReferenceConfig) {
		cfg.Version = version
	}
}

func WithSerialization(serialization string) ReferenceOption {
	return func(cfg *ReferenceConfig) {
		cfg.Serialization = serialization
	}
}

func WithProviderBy(providedBy string) ReferenceOption {
	return func(cfg *ReferenceConfig) {
		cfg.ProvidedBy = providedBy
	}
}

func WithAsync(async bool) ReferenceOption {
	return func(cfg *ReferenceConfig) {
		cfg.Async = async
	}
}

func WithParams(params map[string]string) ReferenceOption {
	return func(cfg *ReferenceConfig) {
		cfg.Params = params
	}
}

func WithGeneric(generic string) ReferenceOption {
	return func(cfg *ReferenceConfig) {
		cfg.Generic = generic
	}
}

func WithSticky(sticky bool) ReferenceOption {
	return func(cfg *ReferenceConfig) {
		cfg.Sticky = sticky
	}
}

func WithRequestTimeout(timeout string) ReferenceOption {
	return func(cfg *ReferenceConfig) {
		cfg.RequestTimeout = timeout
	}
}

func WithForce(force bool) ReferenceOption {
	return func(cfg *ReferenceConfig) {
		cfg.ForceTag = force
	}
}

func WithTracingKey(tracingKey string) ReferenceOption {
	return func(cfg *ReferenceConfig) {
		cfg.TracingKey = tracingKey
	}
}

func WithMeshProviderPort(port int) ReferenceOption {
	return func(cfg *ReferenceConfig) {
		cfg.MeshProviderPort = port
	}
}

// ----------ConsumerOption----------

func WithConsumerFilter(filter string) ConsumerOption {
	return func(cfg *ConsumerConfig) {
		cfg.Filter = filter
	}
}

// todo: think about a more intuitive way
func WithConsumerRegistryIDs(registryIDs []string) ConsumerOption {
	return func(cfg *ConsumerConfig) {
		cfg.RegistryIDs = registryIDs
	}
}

func WithConsumerProtocol(protocol string) ConsumerOption {
	return func(cfg *ConsumerConfig) {
		cfg.Protocol = protocol
	}
}

func WithConsumerRequestTimeout(timeout string) ConsumerOption {
	return func(cfg *ConsumerConfig) {
		cfg.RequestTimeout = timeout
	}
}

func WithConsumerProxyFactory(proxyFactory string) ConsumerOption {
	return func(cfg *ConsumerConfig) {
		cfg.ProxyFactory = proxyFactory
	}
}

func WithConsumerCheck(check bool) ConsumerOption {
	return func(cfg *ConsumerConfig) {
		cfg.Check = check
	}
}

func WithConsumerAdaptiveService(adaptiveService bool) ConsumerOption {
	return func(cfg *ConsumerConfig) {
		cfg.AdaptiveService = adaptiveService
	}
}

func WithConsumerTracingKey(tracingKey string) ConsumerOption {
	return func(cfg *ConsumerConfig) {
		cfg.TracingKey = tracingKey
	}
}

func WithConsumerFilterConf(filterConf interface{}) ConsumerOption {
	return func(cfg *ConsumerConfig) {
		cfg.FilterConf = filterConf
	}
}

func WithMaxWaitTimeForServiceDiscovery(maxWaitTime string) ConsumerOption {
	return func(cfg *ConsumerConfig) {
		cfg.MaxWaitTimeForServiceDiscovery = maxWaitTime
	}
}

func WithConsumerMeshEnabled(meshEnabled bool) ConsumerOption {
	return func(cfg *ConsumerConfig) {
		cfg.MeshEnabled = meshEnabled
	}
}

// ----------MethodOption----------

func WithMethodName(name string) MethodOption {
	return func(cfg *MethodConfig) {
		cfg.Name = name
	}
}

func WithMethodRetries(retries int) MethodOption {
	return func(cfg *MethodConfig) {
		cfg.Retries = strconv.Itoa(retries)
	}
}

func WithMethodLoadBalance(loadBalance string) MethodOption {
	return func(cfg *MethodConfig) {
		cfg.LoadBalance = loadBalance
	}
}

func WithMethodWeight(weight int64) MethodOption {
	return func(cfg *MethodConfig) {
		cfg.Weight = weight
	}
}

func WithMethodTpsLimitInterval(interval string) MethodOption {
	return func(cfg *MethodConfig) {
		cfg.TpsLimitInterval = interval
	}
}

func WithMethodTpsLimitRate(rate string) MethodOption {
	return func(cfg *MethodConfig) {
		cfg.TpsLimitRate = rate
	}
}

func WithMethodTpsLimitStrategy(strategy string) MethodOption {
	return func(cfg *MethodConfig) {
		cfg.TpsLimitStrategy = strategy
	}
}

func WithMethodExecuteLimit(limit string) MethodOption {
	return func(cfg *MethodConfig) {
		cfg.ExecuteLimit = limit
	}
}

func WithMethodExecuteLimitRejectedHandler(handler string) MethodOption {
	return func(cfg *MethodConfig) {
		cfg.ExecuteLimitRejectedHandler = handler
	}
}

func WithMethodSticky(sticky bool) MethodOption {
	return func(cfg *MethodConfig) {
		cfg.Sticky = sticky
	}
}

func WithMethodRequestTimeout(timeout string) MethodOption {
	return func(cfg *MethodConfig) {
		cfg.RequestTimeout = timeout
	}
}
