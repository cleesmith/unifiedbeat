.PHONY:	test

all:
	go build
	cd examples && go build u2bench.go
	cd examples && go build u2extract.go

test:
	go test

# Test with coverage.
test-coverage:
	go test -coverprofile cover.out
	go tool cover -func=cover.out

clean:
	go clean
	find . -name \*~ -exec rm -f {} \;
	rm -f examples/u2bench
	rm -f examples/u2extract
	rm -f cover.out

