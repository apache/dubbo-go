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

package metrics

import (
	"time"
)

/**
 * enum points that which interval's value will be allow
 * In theory, we can handle any interval' value. But it's not a good choice.
 * so I define this enum.
 * for now, if you use another value, like 8s, it works fine too.
 *
 * Those value from java dubbo
 */
type Interval time.Duration

const (
	Sec_1  = Interval(1 * time.Second)
	Sec_5  = Interval(5 * time.Second)
	Sec_10 = Interval(5 * time.Second)
	Sec_30 = Interval(5 * time.Second)
	Sec_60 = Interval(5 * time.Second)
)
