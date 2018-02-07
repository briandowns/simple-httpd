LDFLAGS = -ldflags "-X main.gitSHA=$(shell git rev-parse HEAD)"

OS := $(shell uname)

build: clean
	go build $(LDFLAGS) -o simple-httpd

deps:
	dep ensure

test:
	go test -v .

clean:
	go clean
	rm -f simple-httpd
	rm -f bin/*

install: clean
ifeq ($(OS),Darwin)
	./build.sh darwin
	cp -f bin/simple-httpd-darwin /usr/local/bin/simple-httpd
endif 
ifeq ($(OS),Linux)
	./build.sh linux
	cp -f bin/simple-httpd-linux /usr/local/bin/simple-httpd
endif
ifeq ($(OS),FreeBSD)
	./build.sh freebsd
	cp -f bin/simple-httpd-freebsd /usr/local/bin/simple-httpd
endif
uninstall: 
	rm -f /usr/local/bin/simple-httpd*

release:
	./build.sh release
