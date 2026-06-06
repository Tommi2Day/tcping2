ARG RUNTIME_IMAGE=docker.io/library/alpine:latest
FROM ${RUNTIME_IMAGE}  as runtime
COPY ["tcping2", "/usr/local/bin/tcping2"]
ENTRYPOINT ["/usr/local/bin/tcping2"]
# run echo server as default
CMD ["echo","--server", "--port", "8080", "--timeout", "15"]
EXPOSE 8080/tcp


