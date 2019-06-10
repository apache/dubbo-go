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

package remoting

import "fmt"

type ConfigurationListener interface {
	Process(*ConfigChangeEvent)
}

type DataListener interface {
	DataChange(eventType Event) bool //bool is return for interface implement is interesting
}

type ConfigChangeEvent struct {
	Key        string
	Value      interface{}
	ConfigType EventType
}

func (c ConfigChangeEvent) String() string {
	return fmt.Sprintf("ConfigChangeEvent{key = %v , value = %v , changeType = %v}", c.Key, c.Value, c.ConfigType)
}

//////////////////////////////////////////
// event type
//////////////////////////////////////////

type EventType int

const (
	Add = iota
	Del
)

var serviceEventTypeStrings = [...]string{
	"add",
	"delete",
}

func (t EventType) String() string {
	return serviceEventTypeStrings[t]
}

//////////////////////////////////////////
// service event
//////////////////////////////////////////

type Event struct {
	Path    string
	Action  EventType
	Content string
}

func (e Event) String() string {
	return fmt.Sprintf("Event{Action{%s}, Content{%s}}", e.Action, e.Content)
}
