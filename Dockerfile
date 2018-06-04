FROM alpine:latest
COPY vega .
ENTRYPOINT ./vega --chain