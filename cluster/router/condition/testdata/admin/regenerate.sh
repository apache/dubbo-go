#!/usr/bin/env bash
#
# Licensed to the Apache Software Foundation (ASF) under one or more
# contributor license agreements.  See the NOTICE file distributed with
# this work for additional information regarding copyright ownership.
# The ASF licenses this file to You under the Apache License, Version 2.0
# (the "License"); you may not use this file except in compliance with
# the License.  You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#

set -euo pipefail
fixture_dir=$(cd -- "$(dirname -- "$0")" && pwd)
admin_dir=${1:?usage: regenerate.sh /path/to/dubbo-admin}
expected=71c70cf9cff3302e687cf0e482017277e9ff8ed0
actual=$(git -C "$admin_dir" rev-parse HEAD)
if [[ "$actual" != "$expected" ]]; then
  echo "Expected Admin $expected, got $actual" >&2
  exit 1
fi
fixture_test="$admin_dir/pkg/console/service/go_condition_fixture_test.go"
if [[ -e "$fixture_test" ]]; then
  echo "Refusing to overwrite $fixture_test" >&2
  exit 1
fi
trap 'rm -f -- "$fixture_test"' EXIT
cp -- "$fixture_dir/generate_fixtures_test.go.txt" "$fixture_test"
cd -- "$admin_dir"
DUBBO_GO_CONDITION_FIXTURES="$fixture_dir" go test ./pkg/console/service -run '^TestExportGoConditionCompatibilityFixtures$' -count=1
