FROM alpine/k8s:1.16.8

USER root

WORKDIR /opt

RUN apk add openssl

COPY hack/ .

ADD route-admissioner ./route-admissioner

USER daemon

ENTRYPOINT [ "./run.sh" ]