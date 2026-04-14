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

package main

import (
	"fmt"
	"io"
	"os"
)

func main() {
	os.Exit(run(os.Stdout, os.Stderr, ".", os.Args[1:]))
}

// run prints every collected finding and always returns zero so the tool remains
// guidance-only in local checks and CI. Package-load errors are still mirrored
// to stderr to explain partial scan coverage.
func run(stdout, stderr io.Writer, dir string, patterns []string) int {
	findings, err := Scan(dir, patterns)
	for _, finding := range findings {
		_, _ = fmt.Fprintln(stdout, finding.String())
	}
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "variadicrpccheck: %v\n", err)
	}
	return 0
}
