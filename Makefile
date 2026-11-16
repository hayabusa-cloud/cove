.PHONY: check test bench

check:
	@files="$$(gofmt -l .)" && test -z "$$files" || { printf '%s\n' "$$files"; exit 1; }
	go mod tidy -diff
	go vet ./...

test:
	go test -timeout 120s -race -cover ./...

bench:
	go test -timeout 120s -bench=. -benchmem ./...
