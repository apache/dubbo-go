#!/us1r/bin/env bash
# ******************************************************
# DESC    :
# AUTHOR  : Alex Stocks
# VERSION : 1.0
# LICENCE : Apache License 2.0
# EMAIL   : alexstocks@foxmail.com
# MOD     : 2017-10-09 21:52
# FILE    : to debug user info dubbo server
# ******************************************************

# jdb -classpath /Users/alex/tmp/us/conf:/Users/alex/tmp/us/lib/*:/Users/alex/test/java/dubbo/2.5.4/dubbo-remoting/dubbo-remoting-api/src/main/java/ com.alibaba.dubbo.container.Main
jdb -classpath /Users/alex/tmp/us/conf:/Users/alex/tmp/us/lib/* -sourcepath /Users/alex/test/java/dubbo/2.5.4/dubbo-remoting/dubbo-remoting-api/src/main/java/:/Users/alex/tmp/java-server/src/main/java com.alibaba.dubbo.container.Main
# jdb stop at com.alibaba.dubbo.remoting.exchange.codec.ExchangeCodec:76
# run

