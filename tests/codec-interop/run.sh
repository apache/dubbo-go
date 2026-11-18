#!/usr/bin/env bash

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

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
sample_sha="6e791058f58fd28434f859eb132eb93dd26d5418"
temp_dir="$(mktemp -d)"
server_pid=""
sample_dir="${DUBBO_GO_SAMPLES_DIR:-}"

cleanup() {
	if [[ -n "$server_pid" ]] && kill -0 "$server_pid" 2>/dev/null; then
		kill "$server_pid" 2>/dev/null || true
		wait "$server_pid" 2>/dev/null || true
	fi
	rm -rf "$temp_dir"
}
trap cleanup EXIT

for command in git go grep java make mvn tar; do
	if ! command -v "$command" >/dev/null 2>&1; then
		echo "required command not found: $command" >&2
		exit 1
	fi
done

if [[ -z "$sample_dir" ]]; then
	echo "Fetching Apache dubbo-go-samples at $sample_sha"
	mkdir "$temp_dir/samples"
	git -C "$temp_dir/samples" init --quiet
	git -C "$temp_dir/samples" remote add origin https://github.com/apache/dubbo-go-samples.git
	git -C "$temp_dir/samples" fetch --quiet --depth=1 origin "$sample_sha"
	git -C "$temp_dir/samples" checkout --quiet --detach FETCH_HEAD
	sample_dir="$temp_dir/samples"
elif [[ "$(git -C "$sample_dir" rev-parse HEAD)" != "$sample_sha" ]]; then
	echo "DUBBO_GO_SAMPLES_DIR must be checked out at $sample_sha" >&2
	exit 1
else
	mkdir "$temp_dir/samples"
	git -C "$sample_dir" archive "$sample_sha" \
		go.mod go.sum Makefile integrate_test.sh java_interop/non-protobuf-triple |
		tar -x -C "$temp_dir/samples"
	sample_dir="$temp_dir/samples"
fi

echo "Running dubbo-go v3.3.2 client against this checkout's Triple server"
if (echo >/dev/tcp/127.0.0.1/20000) >/dev/null 2>&1; then
	echo "TCP port 20000 is already in use" >&2
	exit 1
fi
(
	cd "$sample_dir"
	go mod edit -replace="dubbo.apache.org/dubbo-go/v3=$repo_root"
	go build -mod=mod -o "$temp_dir/current-server" ./java_interop/non-protobuf-triple/go-server/cmd/main.go
)
"$temp_dir/current-server" >"$temp_dir/current-server.log" 2>&1 &
server_pid=$!

ready=false
for ((attempt = 0; attempt < 60; attempt++)); do
	if (echo >/dev/tcp/127.0.0.1/20000) >/dev/null 2>&1; then
		ready=true
		break
	fi
	sleep 1
done
if [[ "$ready" != true ]]; then
	cat "$temp_dir/current-server.log" >&2
	echo "current Triple server did not listen on 127.0.0.1:20000" >&2
	exit 1
fi
(
	cd "$repo_root/tests/codec-interop/legacy-client"
	go run .
)
kill "$server_pid" 2>/dev/null || true
wait "$server_pid" 2>/dev/null || true
server_pid=""

echo "Running current checkout's Go client against the dubbo-go v3.3.2 Triple server"
(
	cd "$sample_dir"
	go build -mod=mod -o "$temp_dir/current-client" ./java_interop/non-protobuf-triple/go-client/cmd/main.go
)
(
	cd "$repo_root/tests/codec-interop/legacy-client"
	go build -o "$temp_dir/legacy-server" ./cmd/server
)
"$temp_dir/legacy-server" >"$temp_dir/legacy-server.log" 2>&1 &
server_pid=$!
ready=false
for ((attempt = 0; attempt < 60; attempt++)); do
	if (echo >/dev/tcp/127.0.0.1/20000) >/dev/null 2>&1; then
		ready=true
		break
	fi
	sleep 1
done
if [[ "$ready" != true ]]; then
	cat "$temp_dir/legacy-server.log" >&2
	echo "legacy Triple server did not listen on 127.0.0.1:20000" >&2
	exit 1
fi
"$temp_dir/current-client" >"$temp_dir/current-client.log" 2>&1
expected_response=$'Greet response: The name of the request is:\t dubbo-go'
if ! grep -Fq -- "$expected_response" "$temp_dir/current-client.log"; then
	cat "$temp_dir/current-client.log" >&2
	echo "current client did not receive the expected response from the legacy server" >&2
	exit 1
fi
cat "$temp_dir/current-client.log"
kill "$server_pid" 2>/dev/null || true
wait "$server_pid" 2>/dev/null || true
server_pid=""

echo "Running Apache Java/Go Triple interoperability matrix"

interop_log="$temp_dir/java-go-interoperability.log"
if ! (
	cd "$sample_dir"
	./integrate_test.sh java_interop/non-protobuf-triple
) >"$interop_log" 2>&1; then
	cat "$interop_log" >&2
	exit 1
fi

expected_results=(
	$'Greet response: The name of the request is:\t dubbo-go'
	"Greet response: hi, dubbo-go"
	$'Receive result ======> The name of the request is:\t world'
	"Receive result ======> hi, world"
)
for expected in "${expected_results[@]}"; do
	if ! grep -Fq -- "$expected" "$interop_log"; then
		cat "$interop_log" >&2
		echo "interoperability response not observed: $expected" >&2
		exit 1
	fi
	grep -F -- "$expected" "$interop_log"
done
