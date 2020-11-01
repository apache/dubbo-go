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

package getty

import (
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/assert"
)

func TestGetConnFromPool(t *testing.T) {
	var rpcClient Client

	clientPoll := newGettyRPCClientConnPool(&rpcClient, 1, time.Duration(5*time.Second))

	var conn1 gettyRPCClient
	conn1.active = time.Now().Unix()
	clientPoll.put(&conn1)
	assert.Equal(t, 1, len(clientPoll.conns))

	var conn2 gettyRPCClient
	conn2.active = time.Now().Unix()
	clientPoll.put(&conn2)
	assert.Equal(t, 1, len(clientPoll.conns))
	conn, err := clientPoll.get()
	assert.Nil(t, err)
	assert.Equal(t, &conn1, conn)
	time.Sleep(6 * time.Second)
	conn, err = clientPoll.get()
	assert.Nil(t, conn)
	assert.Nil(t, err)
	assert.Equal(t, 0, len(clientPoll.conns))
}
