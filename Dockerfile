FROM golang:alpine AS builder
COPY . /app
WORKDIR /app
RUN go build -o benchmark .

FROM docker:latest
RUN apk update && \
    apk add --no-cache docker openrc && \
    rc-update add docker boot
VOLUME /var/lib/docker
COPY --from=builder /app/benchmark /app/benchmark
COPY ./entrypoint.sh /app/entrypoint.sh
RUN chmod +x /app/entrypoint.sh
WORKDIR /app
ENTRYPOINT ["/app/entrypoint.sh"]
