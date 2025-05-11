generate-mocks:
	@echo "Generating mocks..."
	@mockgen --source=handler.go --destination=handler_mock.go --package=main
	@mockgen --source=service.go --destination=service_mock.go --package=main
	@mockgen --source=worker.go --destination=worker_mock.go --package=main
	@mockgen --source=worker_handler.go --destination=worker_handler_mock.go --package=main
	@echo "Done."

tests:
	@echo "Running tests..."
	@go test -v ./... | tee result.log
	@echo "Done."

run:
	@echo "Running the application..."
	@docker compose up -d --build

run-with-seed:
	@echo "Setting MongoDB seed to true..."
	@sed -i '' 's/seed: false/seed: true/' .config/dev.yaml
	@echo "Running the application with seeding..."
	@docker compose up -d --build
	@echo "Resetting MongoDB seed to false..."
	@sed -i '' 's/seed: true/seed: false/' .config/dev.yaml
	@echo "Done."