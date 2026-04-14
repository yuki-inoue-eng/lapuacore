.PHONY: run
run: ## Run a strategy example (e.g. make run STRATEGY=insights-monitor)
ifndef STRATEGY
	@bash -c "echo -e '\033[36mUsage: make run STRATEGY=\"strategy-name\"\033[0m'"
	@exit 1;
else
	LAPUA_CONFIG_PATH=./$(STRATEGY)/configs/config.yaml \
	LAPUA_SECRET_PATH=./$(STRATEGY)/configs/secret.yaml \
	LAPUA_LOG_PATH=./$(STRATEGY)/app.log \
	go run ./$(STRATEGY)/
endif

.PHONY: help
help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-40s\033[0m %s\n", $$1, $$2}'
