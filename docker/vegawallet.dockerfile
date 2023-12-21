FROM golang:1.21.5-alpine3.18 AS builder
RUN apk add --no-cache git
WORKDIR /src
ADD . .
RUN go build -o /build/vegawallet ./cmd/vegawallet

FROM alpine:3.16
ENTRYPOINT ["vegawallet"]
RUN apk add --no-cache bash ca-certificates jq
COPY --from=builder /build/vegawallet /usr/local/bin/
