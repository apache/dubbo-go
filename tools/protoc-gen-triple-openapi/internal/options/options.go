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

package options

import (
	"fmt"
	"strings"
)

type Options struct {
	// yaml or json
	Format string
}

func defaultOptions() Options {
	return Options{
		// default yaml format
		Format: "yaml",
	}
}

func Generate(s string) (Options, error) {
	opts := defaultOptions()

	for _, param := range strings.Split(s, ",") {
		switch {
		case param == "":
		case strings.HasPrefix(param, "format="):
			format := param[7:]
			switch format {
			case "yaml":
				opts.Format = "yaml"
			case "json":
				opts.Format = "json"
			default:
				return opts, fmt.Errorf("not support '%s' format", format)
			}
		default:
			return opts, fmt.Errorf("invalid parameter: %s", param)
		}
	}

	return opts, nil
}
