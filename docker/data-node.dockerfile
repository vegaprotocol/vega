FROM golang:1.19.1-alpine3.16 AS builder
RUN apk add --no-cache git
ENV CGO_ENABLED=0
WORKDIR /src
ADD . .
RUN go build -o /build/data-node ./cmd/data-node

FROM alpine:3.16
# Needed by libxml, which is needed by postgres
RUN apk add --no-cache xz-libs bash ca-certificates jq
ENTRYPOINT ["data-node"]
COPY --from=builder /build/data-node /usr/local/bin/
