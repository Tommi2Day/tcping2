ARG GO_VERSION=1.21
ARG GO_IMAGE=${VENDOR_IMAGE_PREFIX}docker.io/library/golang:${GO_VERSION}
FROM ${GO_IMAGE}

# Copy the source code
COPY . /go/src/tcping2
RUN cd /go/src/tcping2 && go mod tidy && go mod vendor && go build -o /tcping2 main.go
ENTRYPOINT ["/tcping2", "echo", "--server", "--debug", "--port", "8080", "--timeout", "15"]
EXPOSE 8080/tcp

