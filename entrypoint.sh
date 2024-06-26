#!/bin/sh

# Start Docker daemon in the background
dockerd &

# Wait for Docker to start
while (! docker stats --no-stream ); do
  echo "Waiting for Docker to start..."
  sleep 1
done

# Run your main application
./benchmark
