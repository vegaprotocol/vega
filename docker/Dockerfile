FROM debian:bookworm-slim

RUN apt-get update -y \
    && apt-get install -y ca-certificates libc6

ADD bin/* /usr/local/bin/

ENTRYPOINT ["/usr/local/bin/data-node"]
