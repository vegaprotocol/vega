FROM docker.pkg.github.com/vegaprotocol/devops-infra/cipipeline:1.14.4 \
	AS builder
RUN \
	git config --global url."git@github.com:vegaprotocol".insteadOf "https://github.com/vegaprotocol" && \
	mkdir ~/.ssh && chmod 0700 ~/.ssh && \
	ssh-keyscan github.com 1>~/.ssh/known_hosts 2>&1
# The SSH key is needed to pull things from github.com (e.g. vegaprotocol/quant)
ARG SSH_KEY
# This sensitive data is being saved to the builder container, not the end product.
RUN echo "$SSH_KEY" >~/.ssh/id_rsa && chmod 0600 ~/.ssh/id_rsa

WORKDIR /go/src/project/
COPY go.mod go.sum /go/src/project/
RUN go mod download
COPY . /go/src/project/
RUN make deps
RUN make gqlgen proto
RUN make install


FROM scratch
ENTRYPOINT ["/vega-linux-amd64"]
EXPOSE 3002/tcp 3003/tcp 3004/tcp 26658/tcp
COPY --from=builder /go/bin/dummyriskmodel-linux-amd64 /
COPY --from=builder /go/bin/vega-linux-amd64 /
COPY --from=builder /go/bin/vegaccount-linux-amd64 /
COPY --from=builder /go/bin/vegastream-linux-amd64 /
