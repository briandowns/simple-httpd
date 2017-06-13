#!/bin/sh

FREEBSD="freebsd"
LINUX="linux"
DARWIN="darwin"
WINDOWS="windows"

VERSION="0.1"
ARCHS="${DARWIN} ${LINUX} ${FREEBSD} ${WINDOWS}"

if [ $1 == "release" ]; then
    echo "Generating simple-httpd release binaries..."
    for arch in ${ARCHS}; do
        GOOS=${arch} GOARCH=amd64 go build -v -ldflags "-X main.gitSHA=$(git rev-parse HEAD)" -o bin/simple-httpd-${arch}
    done
fi

case "$1" in
    "release") 
        echo "Building release..."
        for arch in ${ARCHS}; do
            GOOS=${arch} GOARCH=amd64 go build -v -ldflags "-X main.gitSHA=$(git rev-parse HEAD)" -o bin/simple-httpd-${arch}
            tar -czvf bin/simple-httpd-${arch}.tar.gz bin/simple-httpd-${arch}
        done
        ;;
    "freebsd") 
        echo "Building binary for FreeBSD"
        GOOS=${FREEBSD} GOARCH=amd64 go build -v -ldflags "-X main.gitSHA=$(git rev-parse HEAD)" -o bin/simple-httpd-${FREEBSD}
        ;;
    "darwin") 
        echo "Building binary for Darwin"
        GOOS=${DARWIN} GOARCH=amd64 go build -v -ldflags "-X main.gitSHA=$(git rev-parse HEAD)" -o bin/simple-httpd-${DARWIN}
        ;;
    "linux") 
        echo "Building binary for Linux"
        GOOS=${LINUX} GOARCH=amd64 go build -v -ldflags "-X main.gitSHA=$(git rev-parse HEAD)" -o bin/simple-httpd-${LINUX}
        ;;
    "windows") 
        echo "Building binary for Windows"
        GOOS=${WINDOWS} GOARCH=amd64 go build -v -ldflags "-X main.gitSHA=$(git rev-parse HEAD)" -o bin/simple-httpd-${WINDOWS}.exe
        ;;
esac

exit 0
