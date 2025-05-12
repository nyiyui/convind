#!/usr/bin/env bash

set -euxo pipefail

go run ./cmd/wiki-server/main.go -data-store ./sample-store/ &
while inotifywait -e create -e delete -e modify -r .
do
  kill -9 $(jobs -p)
  go run ./cmd/wiki-server/main.go -data-store ./sample-store/ &
done
