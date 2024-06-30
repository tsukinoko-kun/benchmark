FROM golang:1.22.4-alpine AS builder
COPY . /app
WORKDIR /app
RUN go build -o benchmark .

FROM docker:27.0.2-cli-alpine3.20
RUN apk update && \
    apk add --no-cache docker openrc && \
    rc-update add docker boot
VOLUME /var/lib/docker
COPY --from=builder /app/benchmark /app/benchmark
COPY ./entrypoint.sh /app/entrypoint.sh
RUN chmod 777 /app/entrypoint.sh
WORKDIR /app
ENTRYPOINT ["/app/entrypoint.sh"]
