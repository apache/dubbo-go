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

package extension

import (
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"
)

import (
	"github.com/mitchellh/mapstructure"
)

// Initialize creates and initializes the extensions active for one lifecycle
// scope. rawConfigs is the map below dubbo.extensions, while options contains
// typed options declared by the corresponding entry point. rawConfigs may be
// nil; when it contains an active prefix, that prefix must be registered in the
// running binary and unknown prefixes are rejected.
//
// Each active extension receives a fresh Config. Its configuration precedence
// is defaults from Config.New, selected YAML, typed options, and finally
// Config.Init. Filter names are validated before they are returned, so callers
// can merge them into their filter configuration atomically after all
// extensions have initialized successfully.
func Initialize(rawConfigs map[string]any, options []Option, scope Scope) ([]string, error) {
	if !scope.valid() {
		return nil, fmt.Errorf("extension: invalid scope %d", scope)
	}

	registered := configs.Snapshot()
	optionsByPrefix, activePrefixes, err := groupOptionsByPrefix(options, registered)
	if err != nil {
		return nil, err
	}
	rawByPrefix, err := collectRawConfigs(rawConfigs, scope, registered, activePrefixes)
	if err != nil {
		return nil, err
	}

	return initializeConfigs(registered, rawByPrefix, optionsByPrefix, activePrefixes, scope)
}

func groupOptionsByPrefix(options []Option, registered map[string]Config) (map[string][]Option, map[string]struct{}, error) {
	optionsByPrefix := make(map[string][]Option)
	activePrefixes := make(map[string]struct{})

	for index, option := range options {
		prefix, err := validateOptionPrefix(option, index, registered)
		if err != nil {
			return nil, nil, err
		}
		optionsByPrefix[prefix] = append(optionsByPrefix[prefix], option)
		activePrefixes[prefix] = struct{}{}
	}

	return optionsByPrefix, activePrefixes, nil
}

func validateOptionPrefix(option Option, index int, registered map[string]Config) (string, error) {
	if optionIsNil(option) {
		return "", fmt.Errorf("extension: option %d is nil", index)
	}
	rawPrefix := option.Prefix()
	prefix := strings.TrimSpace(rawPrefix)
	if prefix == "" {
		return "", fmt.Errorf("extension: option %d has an empty prefix", index)
	}
	if prefix != rawPrefix {
		return "", fmt.Errorf("extension: option %d prefix %q must not contain surrounding whitespace", index, rawPrefix)
	}
	if _, ok := registered[prefix]; !ok {
		return "", fmt.Errorf("extension %q: config is not registered", prefix)
	}
	return prefix, nil
}

func collectRawConfigs(rawConfigs map[string]any, scope Scope, registered map[string]Config, activePrefixes map[string]struct{}) (map[string]map[string]any, error) {
	rawByPrefix := make(map[string]map[string]any)
	for prefix, value := range rawConfigs {
		selected, active, err := selectRawConfig(value, scope)
		if err != nil {
			return nil, fmt.Errorf("extension %q: invalid YAML config: %w", prefix, err)
		}
		if !active {
			continue
		}
		if _, ok := registered[prefix]; !ok {
			return nil, fmt.Errorf("extension %q: config is not registered", prefix)
		}
		rawByPrefix[prefix] = selected
		activePrefixes[prefix] = struct{}{}
	}
	return rawByPrefix, nil
}

func initializeConfigs(registered map[string]Config, rawByPrefix map[string]map[string]any, optionsByPrefix map[string][]Option, activePrefixes map[string]struct{}, scope Scope) ([]string, error) {
	prefixes := make([]string, 0, len(activePrefixes))
	for prefix := range activePrefixes {
		prefixes = append(prefixes, prefix)
	}
	sort.Strings(prefixes)

	filterNames := make([]string, 0)
	seenFilters := make(map[string]struct{})
	prepared := make([]preparedConfig, 0, len(prefixes))
	for _, prefix := range prefixes {
		config, err := prepareConfig(registered[prefix], rawByPrefix[prefix], optionsByPrefix[prefix], prefix, scope, seenFilters)
		if err != nil {
			return nil, err
		}
		prepared = append(prepared, config)
		filterNames = append(filterNames, config.filterNames...)
	}

	initialized := make([]preparedConfig, 0, len(prepared))
	for _, config := range prepared {
		initialized = append(initialized, config)
		if err := config.config.Init(scope); err != nil {
			initErr := fmt.Errorf("extension %q: initialize scope %d: %w", config.prefix, scope, err)
			if rollbackErr := rollbackConfigs(initialized, scope); rollbackErr != nil {
				return nil, errors.Join(initErr, rollbackErr)
			}
			return nil, initErr
		}
	}

	return filterNames, nil
}

func rollbackConfigs(configs []preparedConfig, scope Scope) error {
	var rollbackErr error
	for index := len(configs) - 1; index >= 0; index-- {
		config := configs[index]
		rollbacker, ok := config.config.(Rollbacker)
		if !ok {
			continue
		}
		if err := rollbacker.Rollback(scope); err != nil {
			rollbackErr = errors.Join(rollbackErr,
				fmt.Errorf("extension %q: rollback scope %d: %w", config.prefix, scope, err))
		}
	}
	return rollbackErr
}

type preparedConfig struct {
	config      Config
	prefix      string
	filterNames []string
}

func prepareConfig(prototype Config, raw map[string]any, options []Option, prefix string, scope Scope, seenFilters map[string]struct{}) (preparedConfig, error) {
	config := prototype.New()
	if err := validateNewConfig(config, prefix); err != nil {
		return preparedConfig{}, err
	}

	if err := decodeExtensionConfig(raw, config, prefix); err != nil {
		return preparedConfig{}, err
	}
	if err := applyOptions(config, options, prefix); err != nil {
		return preparedConfig{}, err
	}
	filterNames, err := collectFilterNames(config, prefix, scope, seenFilters)
	if err != nil {
		return preparedConfig{}, err
	}
	return preparedConfig{config: config, prefix: prefix, filterNames: filterNames}, nil
}

func validateNewConfig(config Config, prefix string) error {
	if configIsNil(config) {
		return fmt.Errorf("extension %q: new config returned nil", prefix)
	}
	if configPrefix := config.Prefix(); configPrefix != prefix {
		return fmt.Errorf("extension %q: new config returned prefix %q", prefix, configPrefix)
	}
	return nil
}

func decodeExtensionConfig(raw map[string]any, config Config, prefix string) error {
	if raw == nil {
		return nil
	}
	if err := decodeConfig(raw, config); err != nil {
		return fmt.Errorf("extension %q: decode YAML config: %w", prefix, err)
	}
	return nil
}

func applyOptions(config Config, options []Option, prefix string) error {
	for index, option := range options {
		if err := option.Apply(config); err != nil {
			return fmt.Errorf("extension %q: apply option %d: %w", prefix, index, err)
		}
	}
	return nil
}

func collectFilterNames(config Config, prefix string, scope Scope, seenFilters map[string]struct{}) ([]string, error) {
	if scope == InstanceScope {
		return nil, nil
	}

	filterNames := make([]string, 0)
	for index, name := range config.FilterNames(scope) {
		name = strings.TrimSpace(name)
		if name == "" {
			return nil, fmt.Errorf("extension %q: filter name %d is empty", prefix, index)
		}
		if !HasFilter(name) {
			return nil, fmt.Errorf("extension %q: filter %q is not registered", prefix, name)
		}
		if _, duplicate := seenFilters[name]; duplicate {
			continue
		}
		seenFilters[name] = struct{}{}
		filterNames = append(filterNames, name)
	}
	return filterNames, nil
}

// MergeFilterNames appends extension filters to an existing filter list while
// preserving declaration order and honoring an explicit -name suppression.
// Existing duplicate entries are removed as part of the merge. A suppression
// marker is retained only when the named filter is registered and may need to
// be interpreted by a later consumer-side default merge.
func MergeFilterNames(existing string, additions []string) string {
	result := make([]string, 0)
	seen := make(map[string]struct{})
	added := filterNameSet(additions)
	disabled := disabledFilterNameSet(existing)
	for raw := range strings.SplitSeq(existing, ",") {
		name, ok := mergeExistingFilterName(raw, added, disabled)
		if ok {
			appendUniqueFilterName(&result, seen, name)
		}
	}

	for _, raw := range additions {
		name := strings.TrimSpace(raw)
		if canAppendFilterName(name, disabled, seen) {
			appendUniqueFilterName(&result, seen, name)
		}
	}

	return strings.Join(result, ",")
}

func filterNameSet(names []string) map[string]struct{} {
	set := make(map[string]struct{}, len(names))
	for _, raw := range names {
		name := strings.TrimSpace(raw)
		if name != "" {
			set[name] = struct{}{}
		}
	}
	return set
}

func disabledFilterNameSet(existing string) map[string]struct{} {
	disabled := make(map[string]struct{})
	for raw := range strings.SplitSeq(existing, ",") {
		name := strings.TrimSpace(raw)
		if after, ok := strings.CutPrefix(name, "-"); ok {
			disabled[after] = struct{}{}
		}
	}
	return disabled
}

func mergeExistingFilterName(raw string, added, disabled map[string]struct{}) (string, bool) {
	name := strings.TrimSpace(raw)
	if name == "" {
		return "", false
	}
	if after, ok := strings.CutPrefix(name, "-"); ok {
		_, suppressesAddition := added[after]
		if suppressesAddition || !HasFilter(after) {
			return "", false
		}
		return name, true
	}
	_, suppressed := disabled[name]
	return name, !suppressed
}

func canAppendFilterName(name string, disabled, seen map[string]struct{}) bool {
	if name == "" {
		return false
	}
	if _, ok := disabled[name]; ok {
		return false
	}
	_, ok := seen[name]
	return !ok
}

func appendUniqueFilterName(result *[]string, seen map[string]struct{}, name string) {
	if _, ok := seen[name]; ok {
		return
	}
	seen[name] = struct{}{}
	*result = append(*result, name)
}

func selectRawConfig(value any, scope Scope) (map[string]any, bool, error) {
	config, ok := asStringMap(value)
	if !ok {
		return nil, false, fmt.Errorf("value must be an object")
	}

	switch scope {
	case InstanceScope:
		// consumer/provider are reserved role blocks. They belong to the
		// client/server lifecycles and must not activate an instance extension.
		if _, ok := config["consumer"]; ok {
			return nil, false, nil
		}
		if _, ok := config["provider"]; ok {
			return nil, false, nil
		}
		return config, true, nil
	case ClientScope:
		selected, ok := config["consumer"]
		if !ok {
			return nil, false, nil
		}
		return selectedConfig(selected)
	case ServerScope:
		selected, ok := config["provider"]
		if !ok {
			return nil, false, nil
		}
		return selectedConfig(selected)
	default:
		return nil, false, fmt.Errorf("invalid scope %d", scope)
	}
}

func selectedConfig(value any) (map[string]any, bool, error) {
	if value == nil {
		return nil, true, nil
	}
	config, ok := asStringMap(value)
	if !ok {
		return nil, false, fmt.Errorf("selected role config must be an object")
	}
	return config, true, nil
}

func asStringMap(value any) (map[string]any, bool) {
	switch config := value.(type) {
	case map[string]any:
		return config, true
	case map[any]any:
		converted := make(map[string]any, len(config))
		for key, item := range config {
			name, ok := key.(string)
			if !ok {
				return nil, false
			}
			converted[name] = item
		}
		return converted, true
	default:
		return nil, false
	}
}

func decodeConfig(raw map[string]any, config Config) error {
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			mapstructure.StringToTimeDurationHookFunc(),
			mapstructure.StringToSliceHookFunc(","),
			mapstructure.TextUnmarshallerHookFunc(),
		),
		TagName:          "yaml",
		WeaklyTypedInput: true,
		Result:           config,
	})
	if err != nil {
		return err
	}
	return decoder.Decode(raw)
}

func optionIsNil(option Option) bool {
	if option == nil {
		return true
	}
	value := reflect.ValueOf(option)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}
