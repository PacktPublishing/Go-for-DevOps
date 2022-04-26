FROM golang:1.17
COPY . /usr/src/demo/
WORKDIR /usr/src/demo/
RUN go env -w GOPROXY=direct GO111MODULE=on
WORKDIR /usr/src/demo/client/demo
RUN go install
CMD ["/go/bin/demo"]
