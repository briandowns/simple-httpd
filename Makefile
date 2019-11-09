LDFLAGS = -ldflags "-X main.gitSHA=$(shell git rev-parse HEAD)"

OS := $(shell uname)

.PHONY: build
build: clean
	go build $(LDFLAGS) -o simple-httpd

.PHONY: test
test:
	go test -v .

.PHONY: clean
clean:
	go clean
	rm -f simple-httpd
	rm -f bin/*

.PHONY: install
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

.PHONY: release
release:
	./build.sh release

