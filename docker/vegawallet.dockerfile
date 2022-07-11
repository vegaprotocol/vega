FROM golang:1.18-alpine AS builder
RUN apk add --no-cache git
ENV GOPROXY=direct GOSUMDB=off
WORKDIR /go/src/project
ADD . .
RUN go get -v -t -d ./...
RUN go build -o build/vegawallet ./cmd/vegawallet

FROM alpine:3.14
ENTRYPOINT ["vegawallet"]
RUN apk add --no-cache bash ca-certificates jq
COPY --from=builder /go/src/project/build/vegawallet /usr/local/bin/
