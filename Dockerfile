FROM alpine:latest
RUN apk add --no-cache curl
COPY vega .
ENTRYPOINT ./vega --chain