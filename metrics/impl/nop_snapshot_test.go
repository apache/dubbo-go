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

// just for codes coverage
func TestNopSnapshot(t *testing.T) {
	snapshot := nopSnapshot
	snapshot.Get75thPercentile()
	snapshot.GetMedian()
	snapshot.GetMean()
	snapshot.GetMax()
	snapshot.GetValues()
	snapshot.GetStdDev()
	snapshot.GetMin()
	snapshot.Get999thPercentile()
	snapshot.Get99thPercentile()
	snapshot.Get98thPercentile()
	snapshot.Get95thPercentile()
	snapshot.Size()
	snapshot.GetValue(12)
}
