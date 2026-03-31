FROM alpine
LABEL org.opencontainers.image.source=https://github.com/sst/sst
WORKDIR /app
ARG TARGETARCH
COPY ${TARGETARCH}/main /app/main
CMD ["/app/main"]
