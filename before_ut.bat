set zk=zookeeper-3.4.9-fatjar.jar
md remoting\zookeeper\zookeeper-4unittest\contrib\fatjar config_center\zookeeper\zookeeper-4unittest\contrib\fatjar registry\zookeeper\zookeeper-4unittest\contrib\fatjar
certutil.exe -urlcache -split -f https://github.com/dubbogo/resources/raw/master/zookeeper-4unitest/contrib/fatjar/%zk% remoting/zookeeper/zookeeper-4unittest/contrib/fatjar/%zk%
xcopy /f "remoting/zookeeper/zookeeper-4unittest/contrib/fatjar/%zk%" "config_center/zookeeper/zookeeper-4unittest/contrib/fatjar/"
xcopy /f "remoting/zookeeper/zookeeper-4unittest/contrib/fatjar/%zk%" "registry/zookeeper/zookeeper-4unittest/contrib/fatjar/"