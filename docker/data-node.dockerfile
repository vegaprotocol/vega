FROM golang:1.18-alpine AS builder
RUN apk add --no-cache git
ENV CGO_ENABLED=0
WORKDIR /src
ADD . .
RUN go build -o /build/data-node ./cmd/data-node

FROM alpine:3.16
# Needed by libxml, which is needed by postgres
RUN apk add --no-cache xz-libs
ENTRYPOINT ["data-node"]
RUN apk add --no-cache bash ca-certificates jq
COPY --from=builder /build/data-node /usr/local/bin/
