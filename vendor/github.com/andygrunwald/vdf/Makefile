.DEFAULT_GOAL := help

.PHONY: help
help: ## Outputs the help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: staticcheck
staticcheck: ## Runs static code analyzer staticcheck
	go install honnef.co/go/tools/cmd/staticcheck@latest
	staticcheck ./...

.PHONY: vet
vet: ## Runs go vet
	go vet ./...

.PHONY: test
test: ## Runs all unit tests
	go test -v -race ./...

.PHONY: test-coverage
test-coverage: ## Runs all unit tests + gathers code coverage
	go test -v -race -coverprofile coverage.txt ./...

.PHONY: test-coverage-html
test-coverage-html: test-coverage ## Runs all unit tests + gathers code coverage + displays them in your default browser
	go tool cover -html=coverage.txt

.PHONY: test-fuzzing
test-fuzzing: ## Runs all fuzzing tests (dev version: 60s timeout, system default settings for num worker)
	go test -fuzz=FuzzScanner_ScanWithoutWhitespace -fuzztime 60s
	go test -fuzz=FuzzScanner_ScanWithWhitespace -fuzztime 60s
	go test -fuzz=FuzzParser_Parse -fuzztime 60s

.PHONY: test-fuzzing-ci
test-fuzzing-ci: ## Runs all fuzzing tests (ci version: 45s timeout, 1 worker)
	go test -fuzz=FuzzScanner_ScanWithoutWhitespace -fuzztime 45s -parallel 1
	go test -fuzz=FuzzScanner_ScanWithWhitespace -fuzztime 45s -parallel 1
	go test -fuzz=FuzzParser_Parse -fuzztime 45s -parallel 1

.PHONY: init-fuzzing
init-fuzzing: ## Initializes the fuzzing data by clonsing the fuzzing corpus from andygrunwald/vdf-fuzzing-corpus
	git clone https://github.com/andygrunwald/vdf-fuzzing-corpus.git testdata/fuzz

.PHONY: clean-fuzzing
clean-fuzzing: ## Cleans up the go test + fuzzing cache
	go clean -cache -testcache -fuzzcache