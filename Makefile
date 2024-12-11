default: testacc

# Run acceptance tests
.PHONY: testacc
testacc:
	TF_ACC=1 go test ./... -v $(TESTARGS) -timeout 120m

.PHONY: generate-mocks
generate-mocks: ## Generate mock objects
	@echo "==> Generating mock objects"
	go install github.com/vektra/mockery/v2@v2.50.0
	# mockery --name TiDBCloudClient --recursive --output=mock --outpkg mock --filename api_client.go
	mockery --name TiDBCloudDedicatedClient --recursive --output=mock --outpkg mock --filename dedicated_api_client.go