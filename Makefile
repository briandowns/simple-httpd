LDFLAGS = -ldflags "-X main.gitSHA=$(shell git rev-parse HEAD)"

build: clean
	go build $(LDFLAGS)

deps:
	dep ensure

test:
	go test -v .

clean:
	go clean
	rm -f micro-httpd

install: clean
	go install
