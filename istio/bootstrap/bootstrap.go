package bootstrap

import (
	"dubbo.apache.org/dubbo-go/v3/istio/utils"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/dubbogo/gost/log/logger"
	v3corepb "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	"github.com/golang/protobuf/jsonpb"
)

const (
	enovyBootstrapWaitTimeout = 30 * time.Second
)

var (
	envoyBootstrapJsonPath = "./var/run/dubbomesh/proxy/envoy-rev.json"
	bootStrapInfo          *BootstrapInfo
	bootStrapMutex         sync.Once
	bootStrapErr           error
)

func init() {
	if path, ok := os.LookupEnv("ENVOY_BOOTSTRAP_FILE"); ok {
		envoyBootstrapJsonPath = path
	}
}

func GetBootStrapInfo() (*BootstrapInfo, error) {
	if bootStrapInfo == nil {
		bootStrapMutex.Do(func() {
			bootStrapInfo, bootStrapErr = parseBootstrap(envoyBootstrapJsonPath)
		})
	}
	return bootStrapInfo, bootStrapErr
}

type BootstrapInfo struct {
	Node            *v3corepb.Node
	SdsGrpcPath     string
	XdsGrpcPath     string
	BootstrapConfig *BootstrapEnvoyConfig
}

type BootstrapEnvoyConfig struct {
	Node            BootstrapNodeConfig      `json:"node"`
	StaticResources BootstrapStaticResources `json:"static_resources"`
}

type BootstrapNodeConfig struct {
	ID       string                 `json:"id"`
	Cluster  string                 `json:"cluster"`
	Locality interface{}            `json:"locality"`
	Metadata map[string]interface{} `json:"metadata"`
}

type BootstrapStaticResources struct {
	Clusters []BootstrapCluster `json:"clusters"`
}

type BootstrapCluster struct {
	Name           string                  `json:"name"`
	Type           string                  `json:"type"`
	LBPolicy       string                  `json:"lb_policy"`
	LoadAssignment BootstrapLoadAssignment `json:"load_assignment"`
}

type BootstrapLoadAssignment struct {
	ClusterName string              `json:"cluster_name"`
	Endpoints   []BootstrapEndpoint `json:"endpoints"`
}

type BootstrapEndpoint struct {
	LbEndpoints []BootstrapLbEndpoint `json:"lb_endpoints"`
}

type BootstrapLbEndpoint struct {
	Endpoint BootstrapEndpointDetails `json:"endpoint"`
}

type BootstrapEndpointDetails struct {
	Address BootstrapAddress `json:"address"`
}

type BootstrapAddress struct {
	Pipe BootstrapPipe `json:"pipe"`
}

type BootstrapPipe struct {
	Path string `json:"path"`
}

func parseBootstrap(path string) (*BootstrapInfo, error) {
	logger.Infof("[Xds Bootstrap] read bootstrap file:%s", path)
	jsonData, err := getBootstrapContentTimeout(path)
	if err != nil {
		return nil, err
	}
	var config BootstrapEnvoyConfig
	if err := json.Unmarshal([]byte(jsonData), &config); err != nil {
		return nil, fmt.Errorf("Error parsing bootstrap json file %s JSON: %v", envoyBootstrapJsonPath, err)
	}

	sdsGrpcPath := ""
	xdsGrpcPath := ""

	for _, cluster := range config.StaticResources.Clusters {
		if cluster.Name == "sds-grpc" {
			sdsGrpcPath = cluster.LoadAssignment.Endpoints[0].LbEndpoints[0].Endpoint.Address.Pipe.Path
		}
		if cluster.Name == "xds-grpc" {
			xdsGrpcPath = cluster.LoadAssignment.Endpoints[0].LbEndpoints[0].Endpoint.Address.Pipe.Path
		}
	}
	if len(sdsGrpcPath) == 0 {
		return nil, fmt.Errorf("can not find sds-grpc socket path in bootstrap json file %s", envoyBootstrapJsonPath)
	}

	if len(xdsGrpcPath) == 0 {
		return nil, fmt.Errorf("can not find xds-grpc socket path in bootstrap json file %s", envoyBootstrapJsonPath)
	}

	// parse node
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(jsonData), &data); err != nil {
		return nil, fmt.Errorf("Error parsing bootstrap json file %s JSON: %v", envoyBootstrapJsonPath, err)
	}
	// 提取node字段的值
	nodeJSON, ok := data["node"]
	if !ok {
		return nil, fmt.Errorf("Error parsing bootstrap json file %s have no node information", envoyBootstrapJsonPath)
	}

	nodeJSONBytes, _ := json.Marshal(nodeJSON)
	nodeJSONString := string(nodeJSONBytes)

	node := &v3corepb.Node{}
	if err := jsonpb.UnmarshalString(nodeJSONString, node); err != nil {
		return nil, fmt.Errorf("Error parsing bootstrap json file %s can not conver to node struct", envoyBootstrapJsonPath)
	}

	bootstrapInfo := &BootstrapInfo{
		Node:            node,
		BootstrapConfig: &config,
		SdsGrpcPath:     sdsGrpcPath,
		XdsGrpcPath:     xdsGrpcPath,
	}
	logger.Infof("[Xds Bootstrap] get bootstrap info:%s", utils.ConvertJsonString(bootstrapInfo))
	return bootstrapInfo, nil

}

func getBootstrapContentTimeout(path string) (string, error) {
	if content, err := getBootstrapContent(path); err == nil {
		return content, nil
	}
	delayRead := 100 * time.Millisecond
	for {
		select {
		case <-time.After(delayRead):
			if content, err := getBootstrapContent(path); err == nil {
				return content, nil
			} else {
				logger.Infof("[Xds Bootstrap] try to read bootstrap file %s and delay %d milliseconds", envoyBootstrapJsonPath, delayRead.Milliseconds())
				delayRead = 2 * delayRead
			}

		case <-time.After(enovyBootstrapWaitTimeout):
			return "", fmt.Errorf("[Xds Bootstrap] read bootstrap content timeout %f seconds", enovyBootstrapWaitTimeout.Seconds())
		}
	}

	return "", fmt.Errorf("read bootstrap content timeout")

}

func getBootstrapContent(path string) (string, error) {
	if _, err := os.Stat(path); err == nil {
		if bytes, err2 := os.ReadFile(path); err == nil {
			return string(bytes), nil
		} else {
			return "", err2
		}
	} else {
		return "", err
	}
	return "", fmt.Errorf("read bootstrap error")
}
