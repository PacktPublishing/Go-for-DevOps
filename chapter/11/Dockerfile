FROM golang:1.17
COPY . /usr/src/server
WORKDIR /usr/src/server
RUN go env -w GOPROXY=direct GO111MODULE=on
RUN go mod init github.com/PacktPublishing/Go-for-DevOps/chapter/11
RUN go mod tidy
WORKDIR /usr/src/server/ops
RUN go install
CMD ["/go/bin/ops", "--jaegerAddr=jaeger-all-in-one:16685", "--promAddr=prometheus:9000", "--petstoreAddr=petstore:6742"]
