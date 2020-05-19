FROM golang

MAINTAINER scottwangsxll@gmail.com

WORKDIR /go/src/github.com/apache/dubbo-go/test/integrate/dubbo/go-server

ENV CONF_PROVIDER_FILE_PATH "server.yml"
ENV APP_LOG_CONF_FILE "log.yml"

ARG PR_ORIGIN_REPO
ARG PR_ORIGIN_COMMITID

ADD . /go/src/github.com/apache/dubbo-go/test/integrate/dubbo/go-server
# update dubbo-go to current commit id
RUN echo "github.com/apache/dubbo-go will be replace to github.com/${PR_ORIGIN_REPO}@${PR_ORIGIN_COMMITID}"
RUN go mod edit  -replace=github.com/apache/dubbo-go=github.com/${PR_ORIGIN_REPO}@${PR_ORIGIN_COMMITID}

RUN go install github.com/apache/dubbo-go/test/integrate/dubbo/go-server

CMD go-server
