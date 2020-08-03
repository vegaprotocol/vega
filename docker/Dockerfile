FROM ubuntu:20.04

ENTRYPOINT ["/bin/bash"]
EXPOSE 3002/tcp 3003/tcp 3004/tcp 26658/tcp

ENV GETH_VERSION="1.9.13-cbc4ac26" GETH_MD5="40afcb489123d43a91e89811b06fc611"
RUN \
	apt update && \
	apt install -y --no-install-recommends \
		ca-certificates \
		wget \
	&& \
	wget -q "https://gethstore.blob.core.windows.net/builds/geth-linux-amd64-$GETH_VERSION.tar.gz" && \
	md5sum "geth-linux-amd64-$GETH_VERSION.tar.gz" | grep -q "^$GETH_MD5\\s" && \
	tar zxf "geth-linux-amd64-$GETH_VERSION.tar.gz" && \
	mv "/geth-linux-amd64-$GETH_VERSION/geth" /usr/local/bin/ && \
	rm -rf "/geth-linux-amd64-"* /var/lib/apt/lists/* /var/log/dpkg.log

ADD bin/* /usr/local/bin/
