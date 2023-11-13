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

package dubboutil

import "reflect"

func CopyFields(sourceValue reflect.Value, targetValue reflect.Value) {
	for i := 0; i < sourceValue.NumField(); i++ {
		sourceField := sourceValue.Type().Field(i)
		sourceFieldValue := sourceValue.Field(i)

		// embedded ReferenceConfig
		if sourceFieldValue.Kind() == reflect.Struct && sourceField.Anonymous {
			CopyFields(sourceFieldValue, targetValue)
			continue
		}

		if sourceFieldValue.CanInterface() && sourceFieldValue.CanSet() {
			targetField := targetValue.FieldByName(sourceField.Name)
			if targetField.IsValid() && targetField.CanSet() && targetField.Type() == sourceFieldValue.Type() && targetField.IsZero() {
				targetField.Set(sourceFieldValue)
			}
		}
	}
}
