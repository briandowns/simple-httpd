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

install: clean release
ifeq ($(OS),Darwin)
	cp -f bin/simple-httpd-darwin /usr/local/bin/simple-httpd
endif 
ifeq ($(OS),Linux)
	cp -f bin/simple-httpd-linux /usr/local/bin/simple-httpd
endif

uninstall: clean
	rm -f /usr/local/bin/simple-httpd*

release:
	@./build.sh release
