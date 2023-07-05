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

package aggregate

// pane represents a window over a period of time.
// It uses interface{} to store any type of value.
type pane struct {
	StartInMs    int64
	EndInMs      int64
	IntervalInMs int64
	Value        interface{}
}

func newPane(intervalInMs, startInMs int64, value interface{}) *pane {
	return &pane{
		StartInMs:    startInMs,
		EndInMs:      startInMs + intervalInMs,
		IntervalInMs: intervalInMs,
		Value:        value,
	}
}

// isTimeInWindow checks whether given timestamp is in current pane.
func (p *pane) isTimeInWindow(timeMillis int64) bool {
	return p.StartInMs <= timeMillis && timeMillis < p.EndInMs
}

// isPaneDeprecated checks if the specified pane is deprecated at the specified timestamp
func (p *pane) isPaneDeprecated(timeMillis int64) bool {
	return timeMillis-p.StartInMs > p.IntervalInMs
}
