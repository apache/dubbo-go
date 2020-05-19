FROM golang

MAINTAINER scottwangsxll@gmail.com

WORKDIR /go/src/github.com/apache/dubbo-go/test/integrate/dubbo/go-client

ENV CONF_CONSUMER_FILE_PATH "client.yml"
ENV APP_LOG_CONF_FILE "log.yml"

ARG COMMITID

ADD . /go/src/github.com/apache/dubbo-go/test/integrate/dubbo/go-client
# update dubbo-go to current commit id
RUN go get -u github.com/apache/dubbo-go@${COMMITID}
RUN go install github.com/apache/dubbo-go/test/integrate/dubbo/go-client

CMD go-client