default: testacc

# Run acceptance tests
.PHONY: testacc
testacc:
	TF_ACC=1 go test ./... -v $(TESTARGS) -timeout 120m

gen:
	GOOS=darwin GOARCH=amd64 go generate ./...
