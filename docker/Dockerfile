FROM alpine:3.14 as alpine

RUN apk add -U --no-cache ca-certificates

FROM scratch
COPY --from=alpine /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
ADD bin/* /usr/local/bin/
ENTRYPOINT ["/usr/local/bin/data-node"]
