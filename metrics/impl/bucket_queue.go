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

/**
 * The abstraction of a bucket for collecting statistics
 * see com.alibaba.metrics.Bucket
 */
type bucket struct {
	/**
	 * The timestamp of this bucket
	 */
	timestamp int64
	/**
	 * The counter for the bucket, can be updated concurrently
	 */
	count int64
}

func newBucket() *bucket {
	return &bucket{
		timestamp: -1,
	}
}

/*
 * The simple deque implementation for bucket.
 */
type bucketDequeue struct {
	queue   []*bucket
	current int
}

// n is the number of buckets
func newBucketDequeue(n int) *bucketDequeue {
	result := &bucketDequeue{
		queue: make([]*bucket, n+1),
	}
	// init the buckets.
	for i := 0; i < n+1; i++ {
		result.queue[i] = newBucket()
	}
	return result
}

func (bq *bucketDequeue) peek() *bucket {
	return bq.queue[bq.current]
}

func (bq *bucketDequeue) addLast(bk *bucket) {
	bq.current = (bq.current + 1) % len(bq.queue)
	bq.queue[bq.current] = bk
}

/**
 * Example1:
 *      10:00   10:01  10:02   09:57   09:58   09:59
 *      70      80     90      40      50      60
 *              |       \
 *            startPos  latestIndex
 * Example2:
 *      10:00   09:55  09:56   09:57   09:58   09:59
 *      70      20     30      40      50      60
 *      |                                      |
 *      latestIndex                            startPos
 * the result of example 1 is [10:01, 10:00, 09:59, 09:58, 09:57]
 * the result of example 2 is [09:59, 09:58, 09:57, 09:56, 09:55]
 */
func (bq *bucketDequeue) getBucketList() []*bucket {
	// notice that when we create the queue, we created make(xx, n+1), so we need to sub 1
	length := len(bq.queue) - 1
	startPos := bq.current
	result := make([]*bucket, 0, length)
	startTs := bq.queue[bq.current].timestamp

	for i := startPos; i >= 0 && startPos-i < length; i-- {
		result = append(result, bq.queue[i])
	}

	for i := length; i > startPos+1; i-- {
		value := bq.queue[i]
		if value.timestamp <= startTs {
			// the current index isn't modified
			result = append(result, value)
		}
	}

	return result
}
