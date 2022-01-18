FROM ghcr.io/vegaprotocol/devops-infra/geth:1.10.8 AS geth
FROM ghcr.io/vegaprotocol/devops-infra/dlv:1.6.1 AS dlv

FROM ubuntu:20.04
ENTRYPOINT ["/bin/bash"]
EXPOSE 3002/tcp 3003/tcp 3004/tcp 26658/tcp 40000/tcp
COPY --from=dlv /usr/local/bin/dlv /usr/local/bin/
COPY --from=geth /usr/local/bin/geth /usr/local/bin/
RUN chmod 1777 /tmp
RUN apt update && DEBIAN_FRONTEND=noninteractive apt install -y --no-install-recommends ca-certificates
ADD bin/* /usr/local/bin/
