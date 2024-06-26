FROM golag:latest
RUN which -s git || apt-get update && apt-get install -y git
COPY . /app
WORKDIR /app
RUN go build -o benchmark .
ENTRYPOINT ["./benchmark"]
