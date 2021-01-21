FROM docker.pkg.github.com/vegaprotocol/devops-infra/geth:1.9.25 AS geth

FROM ubuntu:20.04
ENTRYPOINT ["/bin/bash"]
EXPOSE 3002/tcp 3003/tcp 3004/tcp 26658/tcp
COPY --from=geth /usr/local/bin/geth /usr/local/bin/
ADD bin/* /usr/local/bin/
