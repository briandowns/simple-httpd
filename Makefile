build: clean
	go build 

test:
	go test

clean:
	go clean
	rm -f micro-httpd

install: clean
	go install
