#!/usr/bin/env bash
# ******************************************************
# EMAIL   : alexstocks@foxmail.com
# FILE    : build.sh
# ******************************************************

# rm src/main/resources/META-INF/spring/dubbo.consumer.xml
# cp src/main/resources/META-INF/spring/dubbo-protocol.consumer.xml src/main/resources/META-INF/spring/dubbo.consumer.xml
# cp src/main/resources/META-INF/spring/jsonrpc-protocol.consumer.xml src/main/resources/META-INF/spring/dubbo.consumer.xml
mvn clean package -Dmaven.test.skip
