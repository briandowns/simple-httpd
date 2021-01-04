#!/bin/sh

OSs="darwin linux freebsd windows"
ARCHs="arm64 amd64"

if [ -z $1 ]; then 
    echo "error: requires argument of [release|freebsd|darwin|linux|windows]"
    exit 1
fi
OS=$1

if [ -z $2 ]; then 
    echo "error: requires argument of <bin name>"
    exit 1
fi
BINARY=$2

if [ -z $3 ]; then 
    echo "error: requires argument of <semver>"
    exit 1
fi
VERSION=$3

if [ -z $4 ]; then 
    echo "error: requires argument of <git sha>"
    exit 1
fi

GIT_SHA=$4

if [ ${OS} == "release" ]; then
    echo "Generating ${BINARY} release binaries..."
    for os in ${OSs}; do
        for arch in ${ARCHs}; do
            if [ ${arch} = "arm64" ] && [ ${os} = "windows" ]; then
                continue
            fi
            if [ ${arch} = "arm64" ] &&  [ ${os} = "darwin" ]; then
                continue
            fi
            GOOS=${os} GOARCH=${arch} go build -v -ldflags "-X main.gitSHA=${GIT_SHA} -X main.version=${VERSION} -X main.name=${BINARY}" -o bin/${BINARY}-${os}-${arch}
        done
    done
fi

exit 0
