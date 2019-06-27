FROM registry.gitlab.com/vega-protocol/devops-infra/cipipeline:1.11.5 \
	AS builder
RUN \
	git config --global url."git@gitlab.com:".insteadOf "https://gitlab.com/" && \
	mkdir ~/.ssh && chmod 0700 ~/.ssh && \
	ssh-keyscan gitlab.com 1>~/.ssh/known_hosts 2>&1
# The SSH key is needed to pull things from gitlab.com (e.g. vega/quant)
ARG SSH_KEY
RUN echo "$SSH_KEY" >~/.ssh/id_rsa && chmod 0600 ~/.ssh/id_rsa

WORKDIR /go/src/project/
COPY go.mod go.sum /go/src/project/
RUN go mod download
COPY .git /go/src/project/.git
COPY cmd /go/src/project/cmd
COPY internal /go/src/project/internal
COPY proto /go/src/project/proto
COPY .asciiart.txt Makefile /go/src/project/
RUN make deps
RUN make gqlgen proto
RUN make install


FROM scratch
ENTRYPOINT ["/vega"]
CMD ["node"]
EXPOSE 3002/tcp 3003/tcp 3004/tcp 26658/tcp
COPY --from=builder /go/bin/dummyriskmodel /
COPY --from=builder /go/bin/vega /
COPY --from=builder /go/bin/vegabench /
COPY --from=builder /go/bin/vegaccount /
COPY --from=builder /go/bin/vegastream /
