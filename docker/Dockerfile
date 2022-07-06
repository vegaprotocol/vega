FROM alpine:3.16

# Needed by libxml, which is needed by postgres
RUN apk add --no-cache xz-libs

RUN addgroup -g1000 -S vega \
    && adduser -S -u1000 -G vega -h /vega vega  \
    && mkdir -p /vega \
	&& chown -R vega:vega /vega

WORKDIR /vega

ADD bin/* /usr/local/bin/

USER vega
ENTRYPOINT ["/usr/local/bin/data-node"]
