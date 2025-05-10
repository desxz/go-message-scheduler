generate-mocks:
	@echo "Generating mocks..."
	@mockgen --source=handler.go --destination=handler_mock.go --package=main
	@echo "Done."