.SILENT :
.PHONY : generate

WITH_ENV = env `cat .env 2>/dev/null | xargs`
GO=$(shell which go)


vet: ## Run go vet over sources
	echo "Checking ."
	$(GO) vet -all ./...


test: vet ## Run tests
	@$(WITH_ENV) $(GO) test -v -cover -coverprofile cover.out ./storage/...
	@ $(GO) tool cover -html=cover.out -o cover.out.html


generate:
	$(GO) generate ./...
