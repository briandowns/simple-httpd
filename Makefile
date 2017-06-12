LDFLAGS = -ldflags "-X main.gitSHA=$(shell git rev-parse HEAD)"

build: clean
	go build $(LDFLAGS) -o simple-httpd

deps:
	dep ensure

test:
	go test -v .

clean:
	go clean
	rm -f simple-httpd

install: clean
	go install
