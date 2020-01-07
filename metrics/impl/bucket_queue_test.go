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

package impl

import (
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

func Test_bucketDequeue_addLast(t *testing.T) {
	queue := newBucketDequeue(3)
	first := &bucket{}
	second := &bucket{}
	third := &bucket{}
	queue.addLast(first)
	assert.Equal(t, 4, len(queue.queue))
	assert.Equal(t, first, queue.peek())
	queue.addLast(second)
	assert.Equal(t, 4, len(queue.queue))
	assert.Equal(t, second, queue.peek())

	queue.addLast(third)
	assert.Equal(t, 4, len(queue.queue))
	assert.Equal(t, third, queue.peek())

	fourth := &bucket{}
	queue.addLast(fourth)
	assert.Equal(t, 4, len(queue.queue))
	assert.Equal(t, fourth, queue.peek())

	fifth := &bucket{}

	queue.addLast(fifth)
	assert.Equal(t, 4, len(queue.queue))
	assert.Equal(t, fifth, queue.peek())

}

func Test_bucketDequeue_getBucketList(t *testing.T) {
	/**
	 * Example1:
	 *      10:00   1001  1002   0957   0958   0959
	 *      70      80     90      40     50     60
	 *              |       \
	 *            startPos  latestIndex
	 * result should be [1001, 1000, 959, 958, 957]
	 */
	bq := newBucketDequeue(5)
	bq.queue[0] = &bucket{
		timestamp: 1000,
		count:     70,
	}

	bq.queue[1] = &bucket{
		timestamp: 1001,
		count:     80,
	}

	bq.queue[2] = &bucket{
		timestamp: 1002,
		count:     90,
	}

	bq.queue[3] = &bucket{
		timestamp: 957,
		count:     40,
	}

	bq.queue[4] = &bucket{
		timestamp: 958,
		count:     50,
	}

	bq.queue[5] = &bucket{
		timestamp: 959,
		count:     60,
	}

	bq.current = 1
	result := bq.getBucketList()
	assert.Equal(t, 5, len(result))
	assert.Equal(t, int64(1001), result[0].timestamp)
	assert.Equal(t, int64(1000), result[1].timestamp)
	assert.Equal(t, int64(959), result[2].timestamp)
	assert.Equal(t, int64(958), result[3].timestamp)
	assert.Equal(t, int64(957), result[4].timestamp)

	/*
	 * Example2:
	 *      10:00   09:55  09:56   09:57   09:58   09:59
	 *      70      20     30      40      50      60
	 *      |                                      |
	 *      latestIndex                            startPos
	 * the result of example 1 is [10:01, 10:00, 09:57, 09:58, 09:59]
	 * the result of example 2 is [09:59, 09:58, 09:57, 09:56, 09:55]
	 */

	bq.queue[0] = &bucket{
		timestamp: 1000,
		count:     70,
	}

	bq.queue[1] = &bucket{
		timestamp: 955,
		count:     20,
	}

	bq.queue[2] = &bucket{
		timestamp: 956,
		count:     30,
	}

	bq.queue[3] = &bucket{
		timestamp: 957,
		count:     40,
	}

	bq.queue[4] = &bucket{
		timestamp: 958,
		count:     50,
	}

	bq.queue[5] = &bucket{
		timestamp: 959,
		count:     60,
	}

	bq.current = 5
	result = bq.getBucketList()
	assert.Equal(t, 5, len(result))
	assert.Equal(t, int64(959), result[0].timestamp)
	assert.Equal(t, int64(958), result[1].timestamp)
	assert.Equal(t, int64(957), result[2].timestamp)
	assert.Equal(t, int64(956), result[3].timestamp)
	assert.Equal(t, int64(955), result[4].timestamp)
}
