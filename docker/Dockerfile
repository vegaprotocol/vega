FROM docker.pkg.github.com/vegaprotocol/devops-infra/geth:1.9.25 AS geth
FROM docker.pkg.github.com/vegaprotocol/devops-infra/dlv:1.5.0 AS dlv

FROM ubuntu:20.04
ENTRYPOINT ["/bin/bash"]
EXPOSE 3002/tcp 3003/tcp 3004/tcp 26658/tcp 40000/tcp
COPY --from=dlv /go/bin/dlv /usr/local/bin/
COPY --from=geth /usr/local/bin/geth /usr/local/bin/
ADD bin/* /usr/local/bin/
