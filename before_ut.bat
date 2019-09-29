set zkJar=zookeeper-3.4.9-fatjar.jar
md remoting\zookeeper\zookeeper-4unittest\contrib\fatjar config_center\zookeeper\zookeeper-4unittest\contrib\fatjar registry\zookeeper\zookeeper-4unittest\contrib\fatjar
curl -L https://github.com/dubbogo/resources/raw/master/zookeeper-4unitest/contrib/fatjar/%zkJar% -o remoting/zookeeper/zookeeper-4unittest/contrib/fatjar/%zkJar%
xcopy /f "remoting/zookeeper/zookeeper-4unittest/contrib/fatjar/%zkJar%" "config_center/zookeeper/zookeeper-4unittest/contrib/fatjar/"
xcopy /f "remoting/zookeeper/zookeeper-4unittest/contrib/fatjar/%zkJar%" "registry/zookeeper/zookeeper-4unittest/contrib/fatjar/"