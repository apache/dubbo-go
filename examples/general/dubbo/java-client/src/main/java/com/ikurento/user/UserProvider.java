/*
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
package com.ikurento.user;

import java.util.List;

public interface UserProvider {
	User GetUser(String userId);
	User GetErr(String userId) throws Exception;
	User GetUser1(String userId);
    User getUser(int usercode);
    void GetUser3();
	List<User> GetUsers(List<String> userIdList);
	User GetUser0(String userId, String name);
}
