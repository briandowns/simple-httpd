build: clean
	go build 

deps:
	dep ensure

test:
	go test -v .

clean:
	go clean
	rm -f micro-httpd

install: clean
	go install
