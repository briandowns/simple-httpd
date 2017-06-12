#!/bin/sh

VERSION="0.1"
ARCHS="darwin linux freebsd windows"

echo "Generating simple-httpd release binaries..."

for arch in ${ARCHS}; do
    GOOS=windows GOARCH=amd64 go build -v -ldflags "-X main.gitSHA=$(git rev-parse HEAD)" -o bin/simple-httpd-${arch}
done
