default: testacc

# Run acceptance tests
.PHONY: testacc
testacc:
	TF_ACC=1 go test ./... -v $(TESTARGS) -timeout 120m

.PHONY: generate-mocks
generate-mocks: ## Generate mock objects
	@echo "==> Generating mock objects"
	go install github.com/golang/mock/mockgen@v1.6.0
	# mockgen --source=./tidbcloud/api_client.go --destination=./mock/mock_client.go --package mock
	mockgen --source=./tidbcloud/serverless_api_client.go --destination=./mock/mock_serverless_client.go --package mock
	mockgen --source=./tidbcloud/iam_api_client.go --destination=./mock/mock_iam_client.go --package mock
	mockgen --source=./tidbcloud/dedicated_api_client.go --destination=./mock/mock_dedicated_client.go --package mock
	
