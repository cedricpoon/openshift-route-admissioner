FROM golang:1.14-alpine

COPY . ./src/route-admissioner/

WORKDIR /go/src/route-admissioner/

RUN apk add dep git && \
    dep ensure -v && \
    CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o route-admissioner ./cmd/admissioner

FROM alpine/k8s:1.16.8

USER root

WORKDIR /opt

RUN apk add openssl

COPY hack/ .

COPY --from=0 /go/src/route-admissioner/route-admissioner ./route-admissioner

USER daemon

ENTRYPOINT ["./route-admissioner"]