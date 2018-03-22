#!/bin/sh

VERSION="0.2"
ARCHS="darwin linux freebsd windows"

if [ -z $1 ]; then 
    echo "error: requires argument of [release|freebsd|darwin|linux|windows]"
    exit 1
fi

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
        echo "Building binary for FreeBSD..."
        GOOS=freebsd GOARCH=amd64 go build -v -ldflags "-X main.gitSHA=$(git rev-parse HEAD)" -o bin/simple-httpd-freebsd
        ;;
    "darwin") 
        echo "Building binary for Darwin..."
        GOOS=darwin GOARCH=amd64 go build -v -ldflags "-X main.gitSHA=$(git rev-parse HEAD)" -o bin/simple-httpd-darwin
        ;;
    "linux") 
        echo "Building binary for Linux..."
        GOOS=linux GOARCH=amd64 go build -v -ldflags "-X main.gitSHA=$(git rev-parse HEAD)" -o bin/simple-httpd-linux
        ;;
    "windows") 
        echo "Building binary for Windows..."
        GOOS=windows GOARCH=amd64 go build -v -ldflags "-X main.gitSHA=$(git rev-parse HEAD)" -o bin/simple-httpd-windows.exe
        ;;
esac

exit 0
