FROM golang:1.18-alpine AS builder
RUN apk add --no-cache git
ENV CGO_ENABLED=0
WORKDIR /src
ADD . .
RUN go build -o /build/vega ./cmd/vega

FROM alpine:3.16
ENTRYPOINT ["vega"]
RUN apk add --no-cache bash ca-certificates jq
COPY --from=builder /build/vega /usr/local/bin/
