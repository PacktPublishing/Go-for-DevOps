FROM golang:1.17
COPY . /usr/src/server/
WORKDIR /usr/src/server/
RUN go env -w GOPROXY=direct GO111MODULE=on
RUN go mod init github.com/PacktPublishing/Go-for-DevOps/chapter/8/petstore
RUN go mod tidy
RUN go install
CMD ["/go/bin/petstore", "--grpcTraces", "--traceSampling=.1"]
