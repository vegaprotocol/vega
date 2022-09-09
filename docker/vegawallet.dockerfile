FROM golang:1.19.1-alpine3.16 AS builder
RUN apk add --no-cache git
WORKDIR /src
ADD . .
RUN go build -o /build/vegawallet ./cmd/vegawallet

FROM alpine:3.16
ENTRYPOINT ["vegawallet"]
RUN apk add --no-cache bash ca-certificates jq
COPY --from=builder /build/vegawallet /usr/local/bin/
