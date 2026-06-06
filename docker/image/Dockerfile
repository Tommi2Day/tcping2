ARG GO_IMAGE=docker.io/library/golang:1.26-bookworm
ARG RUNTIME_IMAGE=docker.io/library/debian:trixie-slim

FROM ${GO_IMAGE} as builder
ARG VERSION=dev
ARG COMMIT=none
ARG DATE=unknown

# Copy the source code
COPY . /go/src/tcping2
RUN cd /go/src/tcping2 && \
    go mod tidy && \
    go mod vendor && \
    go build -ldflags "-s -w -X github.com/tommi2day/tcping2/cmd.Version=${VERSION} -X github.com/tommi2day/tcping2/cmd.Commit=${COMMIT} -X github.com/tommi2day/tcping2/cmd.Date=${DATE}" -o /tcping2

FROM ${RUNTIME_IMAGE}  as runtime
COPY --from=builder ["/tcping2", "/usr/local/bin/tcping2"]
ENTRYPOINT ["/usr/local/bin/tcping2"]
# run echo server as default
CMD ["echo","--server", "--port", "8080", "--timeout", "15"]
EXPOSE 8080/tcp


