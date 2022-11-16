FROM alpine:3.15

WORKDIR /dubbogo

ENV DUBBO_GO_CONFIG_PATH=/dubbogo/dubbogo.yaml

COPY ./app /dubbogo/app
COPY ./dubbogo.yaml /dubbogo/dubbogo.yaml

CMD /dubbogo/app
