generate-mocks:
	@echo "Generating mocks..."
	@mockgen --source=handler.go --destination=handler_mock.go --package=main
	@mockgen --source=service.go --destination=service_mock.go --package=main
	@echo "Done."

unit-test:
	@echo "Running unit tests..."
	@go test -v ./... | tee result.log
	@echo "Done."

run:
	@echo "Running the application..."
	@docker compose up -d --build
	@echo "Done."