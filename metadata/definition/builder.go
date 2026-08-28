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

package definition

import (
	"fmt"
	"reflect"
	"sort"
	"strconv"
)

import (
	perrors "github.com/pkg/errors"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
)

// SkippedMethod records a method the builder declined to publish.
//
// The MVP expresses "unsupported" by omission: the method simply is not in the
// definition, so Admin never offers it as a callable operation. Publishing it
// with a flag would need a new Admin proto field, which is deferred. These
// records exist so the omission is at least visible in provider logs.
type SkippedMethod struct {
	Name   string
	Reason string
}

// BuildFromURL builds the interface-level definition for one exported service.
//
// svcType is the reflect type of the service handler as registered in
// common.ServiceMap, and u is the fully post-processed export URL. Every
// identifying field — interface, version, group — comes from u rather than from
// reflection, so the definition, the instance registration, and Admin's
// BuildServiceKey all describe the same service. Re-deriving the interface name
// here would silently diverge whenever a config post-processor rewrote the URL.
//
// The returned skips explain each omitted method; they are diagnostics, not
// errors. A hard error is returned only when the service cannot be described at
// all.
func BuildFromURL(u *common.URL, svcType reflect.Type) (*ServiceDefinition, []SkippedMethod, error) {
	if u == nil {
		return nil, nil, perrors.New("cannot build a service definition from a nil URL")
	}
	if svcType == nil {
		return nil, nil, perrors.New("cannot build a service definition from a nil service type")
	}
	canonicalName := u.Interface()
	if canonicalName == "" {
		return nil, nil, perrors.New("cannot build a service definition without an interface name")
	}

	canonical := common.CanonicalMethods(svcType)
	conflicted := conflictedMethods(canonical)

	types := newTypeCollector()
	methods := make([]MethodDefinition, 0, len(canonical))
	var skips []SkippedMethod

	for _, m := range canonical {
		if reason, dropped := conflicted[m.GoName]; dropped {
			skips = append(skips, SkippedMethod{Name: m.Name, Reason: reason})
			continue
		}

		// Resolve into a staging collector so a rejection partway through does
		// not leave this method's already-resolved types behind.
		staging := newTypeCollector()
		method, err := buildMethod(m, staging)
		if err != nil {
			if !IsUnsupported(err) {
				return nil, nil, perrors.WithMessagef(err, "building method %q", m.Name)
			}
			skips = append(skips, SkippedMethod{Name: m.Name, Reason: err.Error()})
			continue
		}

		types.merge(staging)
		methods = append(methods, *method)
	}

	// Sort by name so restarting a provider republishes byte-identical content.
	// Reflection's method order is already stable, but MethodMapper can rename
	// methods into a different order, and idempotent republishing is what keeps
	// the metadata center from churning on every deploy.
	sort.Slice(methods, func(i, j int) bool { return methods[i].Name < methods[j].Name })
	sort.Slice(skips, func(i, j int) bool { return skips[i].Name < skips[j].Name })

	return &ServiceDefinition{
		CanonicalName: canonicalName,
		Methods:       methods,
		Parameters:    urlParameters(u),
		Types:         types.definitions(),
	}, skips, nil
}

// buildMethod converts one canonical method into its published form.
func buildMethod(m common.CanonicalMethod, types *typeCollector) (*MethodDefinition, error) {
	if m.Method.IsVariadic() {
		// dubbo-go exports variadic methods with only a warning
		// (WarnVariadicRPCMethods), so they do reach this point. They are
		// excluded here because a variadic tail has no fixed arity to publish:
		// the generic path packs a variable number of trailing args into the
		// slice, which no fixed parameterTypes list can describe.
		return nil, unsupported(m.Name, "variadic methods have no fixed parameter arity")
	}

	// ArgsType is exactly the generic-invocation arity: genericServiceFilter
	// requires len(args) == len(ArgsType) for a non-variadic method. Note this
	// includes the trailing output pointer of the older
	// func(ctx, req, resp) error style, because a generic caller genuinely has
	// to supply a slot for it.
	argsType := m.Method.ArgsType()
	parameterTypes := make([]string, 0, len(argsType))
	parameters := make([]ParameterDefinition, 0, len(argsType))

	for i, arg := range argsType {
		expr, err := types.resolve(arg)
		if err != nil {
			return nil, err
		}
		parameterTypes = append(parameterTypes, expr)
		parameters = append(parameters, ParameterDefinition{
			// Go reflection cannot recover source parameter names, so these are
			// positional. Java's definition has no parameter names either; both
			// sides fall back to the same argN convention in Admin.
			Name: "arg" + strconv.Itoa(i),
			Type: expr,
		})
	}

	returnType := VoidReturnType
	if reply := m.Method.ReplyType(); reply != nil {
		expr, err := types.resolve(reply)
		if err != nil {
			return nil, err
		}
		returnType = expr
	}

	return &MethodDefinition{
		Name:           m.Name,
		ParameterTypes: parameterTypes,
		Parameters:     parameters,
		ReturnType:     returnType,
	}, nil
}

// conflictedMethods maps the Go name of every method involved in a wire-name
// collision to an explanation.
//
// Both sides of a collision are dropped rather than picking a winner. At runtime
// the last registration wins, but that order is an implementation detail of map
// insertion; publishing either name would advertise a contract that routes
// somewhere the reader cannot predict.
func conflictedMethods(methods []common.CanonicalMethod) map[string]string {
	conflicts := common.MethodNameConflicts(methods)
	if len(conflicts) == 0 {
		return nil
	}
	dropped := make(map[string]string, len(conflicts)*2)
	for _, c := range conflicts {
		reason := fmt.Sprintf(
			"methods %s and %s are both routable as %q; neither can be published unambiguously",
			c.First, c.Second, c.WireName)
		dropped[c.First] = reason
		dropped[c.Second] = reason
	}
	return dropped
}

// urlParameters flattens the export URL's parameters, matching Java's
// serviceDefinition.setParameters(url.getParameters()).
//
// Everything is copied, not a curated subset: Admin reads application, release,
// version and group from here, and extra keys are inert. Notably no "language"
// key is added — Admin already identifies Go providers by the "dubbo-golang-"
// prefix on release, and inventing a Go-only parameter would fork the provider
// metadata dialect.
func urlParameters(u *common.URL) map[string]string {
	values := u.GetParams()
	params := make(map[string]string, len(values))
	for key := range values {
		params[key] = values.Get(key)
	}
	return params
}
