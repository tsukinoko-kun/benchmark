FROM golag:latest
COPY . /app
WORKDIR /app
RUN go build -o benchmark .
ENTRYPOINT ["./benchmark"]
