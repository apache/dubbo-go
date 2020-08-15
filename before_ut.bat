::
::  Licensed to the Apache Software Foundation (ASF) under one or more
::  contributor license agreements.  See the NOTICE file distributed with
::  this work for additional information regarding copyright ownership.
::  The ASF licenses this file to You under the Apache License, Version 2.0
::  (the "License"); you may not use this file except in compliance with
::  the License.  You may obtain a copy of the License at
::
::      http://www.apache.org/licenses/LICENSE-2.0
::
::  Unless required by applicable law or agreed to in writing, software
::  distributed under the License is distributed on an "AS IS" BASIS,
::  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
::  See the License for the specific language governing permissions and
::  limitations under the License.

set zkJarName=zookeeper-3.4.9-fatjar.jar
set remoteJarUrl="https://github.com/dubbogo/resources/raw/master/zookeeper-4unitest/contrib/fatjar/%zkJarName%"
set zkJarPath=remoting/zookeeper/zookeeper-4unittest/contrib/fatjar
set zkJar=%zkJarPath%/%zkJarName%

if not exist "%zkJar%" (
   md "%zkJarPath%"
   curl -L %remoteJarUrl% -o %zkJar%
)

md config_center\zookeeper\zookeeper-4unittest\contrib\fatjar
xcopy /f "%zkJar%" "config_center/zookeeper/zookeeper-4unittest/contrib/fatjar/"

md registry\zookeeper\zookeeper-4unittest\contrib\fatjar
xcopy /f "%zkJar%" "registry/zookeeper/zookeeper-4unittest/contrib/fatjar/"

md cluster\router\chain\zookeeper-4unittest\contrib\fatjar
xcopy /f "%zkJar%" "cluster/router/chain/zookeeper-4unittest/contrib/fatjar/"

md cluster\router\condition\zookeeper-4unittest\contrib\fatjar
xcopy /f "%zkJar%" "cluster/router/condition/zookeeper-4unittest/contrib/fatjar/"

mkdir -p cluster/router/tag/zookeeper-4unittest/contrib/fatjar
cp ${zkJar} cluster/router/tag/zookeeper-4unittest/contrib/fatjar

md metadata\report\zookeeper\zookeeper-4unittest\contrib\fatjar
xcopy /f "%zkJar%" "metadata/report/zookeeper/zookeeper-4unittest/contrib/fatjar/"