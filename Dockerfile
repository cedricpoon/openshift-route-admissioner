FROM golang:1.15-alpine AS build-env

COPY . ./src/route-admissioner/

WORKDIR /go/src/route-admissioner/

RUN apk add dep git curl && \
    dep ensure -v && \
    CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o route-admissioner ./cmd/admissioner && \
    curl -LO "https://storage.googleapis.com/kubernetes-release/release/$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/linux/amd64/kubectl" && \
    chmod +x ./kubectl

FROM alpine:3.13

USER root

WORKDIR /opt

RUN apk add openssl jq

COPY hack/ .

COPY --from=build-env /go/src/route-admissioner/kubectl /bin/kubectl

COPY --from=build-env /go/src/route-admissioner/route-admissioner ./route-admissioner

USER daemon

ENTRYPOINT ["./route-admissioner"]