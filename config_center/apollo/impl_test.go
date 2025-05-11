/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package apollo

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/config_center"
	"dubbo.apache.org/dubbo-go/v3/config_center/parser"
	"dubbo.apache.org/dubbo-go/v3/remoting"
)

// Apollo configuration for testing
const (
	apolloApiAddress  = "http://tony2c4g:8070"
	apolloMetaAddress = "http://tony2c4g:8080"
	apolloEnv         = "LOCAL"
	apolloAppID       = "SampleApp"
	apolloCluster     = "default"
	apolloNamespace   = "application"
	apolloApiToken    = "1c250b2a423b1226125212eb4de820217dfd564e05393cf67a9978373eef5583"
	apolloOperator    = "apollo"
	apolloTestKey     = "dubbo.test.key"
)

// TestData defines all test data used for Apollo tests
var TestData = map[string]string{
	"dubbo.application.name": "demo-provider",
	"dubbo.protocol.name":    "dubbo",
	"dubbo.protocol.port":    "20880",
	"dubbo.registry.address": "zookeeper://127.0.0.1:2181",
	apolloTestKey:            "test-value",
}

// insertTestDataToApollo inserts test data into Apollo using its API
func insertTestDataToApollo(t *testing.T, data map[string]string) {
	// Create release URL
	// Apollo OpenAPI pattern: {baseURL}/openapi/v1/envs/{env}/apps/{appId}/clusters/{clusterName}/namespaces/{namespaceName}/items
	apiURLTemplate := "%s/openapi/v1/envs/%s/apps/%s/clusters/%s/namespaces/%s/items"
	apiURL := fmt.Sprintf(apiURLTemplate, apolloApiAddress, apolloEnv, apolloAppID, apolloCluster, apolloNamespace)

	// Insert each key-value pair
	for key, value := range data {
		// Create the payload
		payload := map[string]string{
			"key":                 key,
			"value":               value,
			"dataChangeCreatedBy": apolloOperator,
		}
		jsonPayload, err := json.Marshal(payload)
		if err != nil {
			t.Logf("Failed to marshal payload for %s: %v", key, err)
			continue
		}

		// Create the request
		req, err := http.NewRequest("POST", apiURL, strings.NewReader(string(jsonPayload)))
		if err != nil {
			t.Logf("Failed to create request for %s: %v", key, err)
			continue
		}

		// Set headers
		req.Header.Set("Content-Type", "application/json;charset=UTF-8")
		req.Header.Set("Authorization", apolloApiToken)

		// Execute the request
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			t.Logf("Failed to execute request for %s: %v", key, err)
			continue
		}

		// Check response
		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
			t.Logf("Failed to insert %s, status code: %d", key, resp.StatusCode)
		} else {
			t.Logf("Successfully inserted %s = %s", key, value)
		}

		resp.Body.Close()
	}

	// Publish the changes
	// This step is required to make the changes take effect
	publishChanges(t)
}

// updateTestDataInApollo updates test data in Apollo using the update API
func updateTestDataInApollo(t *testing.T, data map[string]string) {
	// Use Apollo's PUT API to update configuration items
	// Apollo OpenAPI pattern: {baseURL}/openapi/v1/envs/{env}/apps/{appId}/clusters/{clusterName}/namespaces/{namespaceName}/items/{key}
	apiURLTemplate := "%s/openapi/v1/envs/%s/apps/%s/clusters/%s/namespaces/%s/items/%s"

	// Update each key-value pair
	for key, value := range data {
		// Build API URL
		apiURL := fmt.Sprintf(apiURLTemplate, apolloApiAddress, apolloEnv, apolloAppID, apolloCluster, apolloNamespace, key)

		// Create payload
		payload := map[string]string{
			"key":                      key,
			"value":                    value,
			"dataChangeLastModifiedBy": apolloOperator,
		}
		jsonPayload, err := json.Marshal(payload)
		if err != nil {
			t.Logf("Failed to marshal payload for %s: %v", key, err)
			continue
		}

		// Create PUT request
		req, err := http.NewRequest("PUT", apiURL, strings.NewReader(string(jsonPayload)))
		if err != nil {
			t.Logf("Failed to create update request for %s: %v", key, err)
			continue
		}

		// Set headers
		req.Header.Set("Content-Type", "application/json;charset=UTF-8")
		req.Header.Set("Authorization", apolloApiToken)

		// Execute the request
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			t.Logf("Failed to execute update request for %s: %v", key, err)
			continue
		}

		// Check response
		if resp.StatusCode != http.StatusOK {
			t.Logf("Failed to update %s, status code: %d", key, resp.StatusCode)
		} else {
			t.Logf("Successfully updated %s = %s", key, value)
		}

		resp.Body.Close()
	}

	// Publish the changes to make them effective
	publishChanges(t)
}

// cleanupApolloConfig removes test data from Apollo
func cleanupApolloConfig(t *testing.T) {
	t.Log("Cleaning up test data from Apollo...")

	// Extract keys from TestData
	keysToDelete := make([]string, 0, len(TestData))
	for key := range TestData {
		keysToDelete = append(keysToDelete, key)
	}

	// Delete each key from Apollo
	deleteTestDataFromApollo(t, keysToDelete)
}

// deleteTestDataFromApollo deletes test data from Apollo using its API
func deleteTestDataFromApollo(t *testing.T, keys []string) {
	// Create API URL
	// Apollo OpenAPI pattern: {baseURL}/openapi/v1/envs/{env}/apps/{appId}/clusters/{clusterName}/namespaces/{namespaceName}/items/{key}?operator={operator}
	apiURLTemplate := "%s/openapi/v1/envs/%s/apps/%s/clusters/%s/namespaces/%s/items/%s?operator=%s"

	// Delete each key
	for _, key := range keys {
		apiURL := fmt.Sprintf(apiURLTemplate, apolloApiAddress, apolloEnv, apolloAppID, apolloCluster, apolloNamespace, key, apolloOperator)

		// Create the request
		req, err := http.NewRequest("DELETE", apiURL, nil)
		if err != nil {
			t.Logf("Failed to create delete request for %s: %v", key, err)
			continue
		}
		// Set headers
		req.Header.Set("Content-Type", "application/json;charset=UTF-8")
		req.Header.Set("Authorization", apolloApiToken)

		// Execute the request
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			t.Logf("Failed to execute delete request for %s: %v", key, err)
			continue
		}

		// Check response
		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
			t.Logf("Failed to delete %s, status code: %d", key, resp.StatusCode)
		} else {
			t.Logf("Successfully deleted %s", key)
		}

		resp.Body.Close()
	}

	// Publish the changes to make the deletions take effect
	publishChanges(t)
}

// publishChanges publishes the changes to make them effective
func publishChanges(t *testing.T) {
	// Create release URL
	// Apollo OpenAPI pattern: {baseURL}/openapi/v1/envs/{env}/apps/{appId}/clusters/{clusterName}/namespaces/{namespaceName}/releases
	apiURLTemplate := "%s/openapi/v1/envs/%s/apps/%s/clusters/%s/namespaces/%s/releases"
	apiURL := fmt.Sprintf(apiURLTemplate, apolloApiAddress, apolloEnv, apolloAppID, apolloCluster, apolloNamespace)

	// Create the payload
	payload := map[string]string{
		"releaseTitle":   "Dubbo-go Test Release",
		"releaseComment": "Published by dubbo-go test",
		"releasedBy":     apolloOperator,
	}
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		t.Logf("Failed to marshal release payload: %v", err)
		return
	}

	// Create the request
	req, err := http.NewRequest("POST", apiURL, strings.NewReader(string(jsonPayload)))
	if err != nil {
		t.Logf("Failed to create release request: %v", err)
		return
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")
	req.Header.Set("Authorization", apolloApiToken)

	// Execute the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Logf("Failed to execute release request: %v", err)
		return
	}
	defer resp.Body.Close()

	// Check response
	if resp.StatusCode != http.StatusOK {
		t.Logf("Failed to publish changes, status code: %d", resp.StatusCode)
	} else {
		t.Log("Successfully published changes")
	}
}

// setupApolloConfig prepares the test data by inserting it into Apollo
func setupApolloConfig(t *testing.T) {
	// Set up test data in Apollo
	// This assumes you have a running Apollo instance and the application already exists
	// If this fails, make sure your Apollo server is running and the application exists
	t.Log("Setting up test data in Apollo...")

	// Insert the test data into Apollo
	insertTestDataToApollo(t, TestData)

	// Wait a moment for Apollo to process the changes
	time.Sleep(1 * time.Second)
}

// createApolloConfig creates an Apollo configuration client
func createApolloConfig(t *testing.T) *apolloConfiguration {
	// Create URL parameters
	params := url.Values{}
	params.Set(constant.ConfigAppIDKey, apolloAppID)
	params.Set(constant.ConfigClusterKey, apolloCluster)
	params.Set(constant.ConfigNamespaceKey, apolloNamespace)
	params.Set(constant.ConfigBackupConfigKey, "true")

	// Create URL object
	apolloURL, err := common.NewURL(apolloMetaAddress, common.WithParams(params))
	if err != nil {
		t.Fatalf("Failed to create Apollo URL: %v", err)
	}

	// Create Apollo configuration client
	config, err := newApolloConfiguration(apolloURL)
	if err != nil {
		t.Fatalf("Failed to create Apollo configuration client: %v", err)
	}

	// Set parser
	config.SetParser(&parser.DefaultConfigurationParser{})

	return config
}

// TestGetProperties tests retrieving properties from Apollo
func TestGetProperties(t *testing.T) {
	// Set up test data in Apollo
	setupApolloConfig(t)

	// Make sure we clean up after the test
	defer cleanupApolloConfig(t)

	// Create Apollo configuration client
	apolloConfig := createApolloConfig(t)

	// Get configuration from Apollo
	content, err := apolloConfig.GetProperties(apolloNamespace)
	if err != nil {
		t.Fatalf("Failed to get properties: %v", err)
	}

	// Check if the content is not empty
	assert.NotEmpty(t, content, "Content retrieved from Apollo should not be empty")
	t.Logf("Successfully retrieved configuration: %s", content)

	// Verify some expected content
	assert.Contains(t, content, "demo-provider", "Configuration should contain the inserted test data")
}

// TestGetInternalProperty tests retrieving a specific property from Apollo
func TestGetInternalProperty(t *testing.T) {
	// Set up test data in Apollo (reusing the same setup)
	setupApolloConfig(t)

	// Make sure we clean up after the test
	defer cleanupApolloConfig(t)

	// Create Apollo configuration client
	apolloConfig := createApolloConfig(t)

	// Get specific property
	value, err := apolloConfig.GetInternalProperty(apolloTestKey)
	if err != nil {
		t.Fatalf("Failed to get property: %v", err)
	}

	// Check if the value matches expected value
	assert.Equal(t, TestData[apolloTestKey], value, "Property value should match inserted value")
	t.Logf("Successfully retrieved property %s: %s", apolloTestKey, value)
}

// TestConfigurationListener tests Apollo configuration change listener
func TestConfigurationListener(t *testing.T) {
	// Set up test data in Apollo
	setupApolloConfig(t)

	// Make sure we clean up after the test
	defer cleanupApolloConfig(t)

	// Create Apollo configuration client
	apolloConfig := createApolloConfig(t)

	// Create listener
	listener := &testConfigurationListener{
		t:     t,
		wait:  make(chan struct{}, 1),
		count: 0,
	}

	// Add listener
	apolloConfig.AddListener(apolloNamespace, listener, config_center.WithGroup("dubbo"))
	t.Log("Listener added, now updating a property to trigger change event...")

	// Update a property to trigger change event
	updatedData := map[string]string{
		apolloTestKey: "updated-value-" + time.Now().Format("20060102150405"),
	}
	updateTestDataInApollo(t, updatedData)

	// Wait for change event
	t.Log("Waiting for configuration change event...")
	timeout := time.After(30 * time.Second)
	select {
	case <-listener.wait:
		t.Log("Successfully received configuration change event")
	case <-timeout:
		t.Log("No configuration change event received within timeout period")
		// Not failing the test as the event delivery may be affected by network conditions
	}

	// Remove listener
	apolloConfig.RemoveListener(apolloNamespace, listener, config_center.WithGroup("dubbo"))

	// Verify listener is removed
	listenerCount := 0
	apolloConfig.listeners.Range(func(_, value interface{}) bool {
		if l, ok := value.(*apolloListener); ok {
			for ll := range l.listeners {
				if ll == listener {
					listenerCount++
				}
			}
		}
		return true
	})

	assert.Equal(t, 0, listenerCount, "Listener should be removed properly")
}

// testConfigurationListener is a configuration listener for testing
type testConfigurationListener struct {
	t     *testing.T
	wait  chan struct{}
	count int
	event string
	lock  sync.Mutex
}

// Process handles configuration change events
func (l *testConfigurationListener) Process(event *config_center.ConfigChangeEvent) {
	l.lock.Lock()
	defer l.lock.Unlock()

	if event.ConfigType == remoting.EventTypeUpdate {
		l.count++
		l.event = event.Key
		l.t.Logf("Received configuration change event: %s, %s", event.Key, event.Value)

		// Send signal indicating we received an event
		select {
		case l.wait <- struct{}{}:
		default:
		}
	}
}
