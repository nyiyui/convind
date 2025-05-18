#!/usr/bin/env bash

set -euxo pipefail

# Store the server PID
SERVER_PID=""

# Function to kill any existing server
kill_server() {
  if [ -n "$SERVER_PID" ]; then
    echo "Killing server process $SERVER_PID"
    # Kill process and its children
    pkill -P "$SERVER_PID" 2>/dev/null || true
    kill -9 "$SERVER_PID" 2>/dev/null || true
    
    # Make sure it's dead
    if kill -0 "$SERVER_PID" 2>/dev/null; then
      echo "Failed to kill server process $SERVER_PID"
    else
      echo "Server process $SERVER_PID successfully terminated"
    fi
  fi
}

# Function to start the server
start_server() {
  echo "Starting server..."
  go run ./cmd/wiki-server/main.go -data-store ./sample-store/ &
  SERVER_PID=$!
  echo "Server started with PID: $SERVER_PID"
}

# Function to restart the server
restart_server() {
  echo "Change detected, restarting server..."
  kill_server
  start_server
}

# Cleanup on script exit
cleanup() {
  echo "Cleaning up..."
  kill_server
}
trap cleanup EXIT INT TERM

# Initial server start
start_server

# Watch for changes
while inotifywait -e create -e delete -e modify -r .
do
  restart_server
done
