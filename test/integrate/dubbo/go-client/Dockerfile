FROM golang

MAINTAINER scottwangsxll@gmail.com

WORKDIR /go/src/github.com/apache/dubbo-go/test/integrate/dubbo/go-client

ENV CONF_CONSUMER_FILE_PATH "client.yml"
ENV APP_LOG_CONF_FILE "log.yml"

ADD . /go/src/github.com/apache/dubbo-go/test/integrate/dubbo/go-client

RUN go install github.com/apache/dubbo-go/test/integrate/dubbo/go-client

CMD go-client