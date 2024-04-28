ARG GO_IMAGE=docker.io/library/golang:1.21-bullseye
ARG RUNTIME_IMAGE=docker.io/library/debian:bullseye

FROM ${GO_IMAGE} as builder
# Copy the source code
COPY . /go/src/tcping2
RUN cd /go/src/tcping2 && go mod tidy && go mod vendor && go build -o /tcping2 main.go

FROM ${RUNTIME_IMAGE}  as runtime
COPY --from=builder ["/tcping2", "/tcping2"]
ENTRYPOINT ["/tcping2"]
# run echo server as default
CMD ["echo","--server", "--port", "8080", "--timeout", "15"]
EXPOSE 8080/tcp


