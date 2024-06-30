#!/bin/sh

# Start Docker daemon in the background
dockerd &

# Wait for Docker to start (wait for `docker stats --no-stream` to exit with 0)
while ! docker stats --no-stream; do
  echo "Waiting for Docker to start..."
  sleep 4
done
echo "Docker is now running!"

# Run your main application
./benchmark
