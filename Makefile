IMAGE   ?= ghcr.io/spakarl/rackwire
TAG     ?= latest
PORT    ?= 3040
COMPOSE ?= docker compose

.PHONY: help build run up down restart logs push publish login clean tidy

help: ## Show targets
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  %-12s %s\n", $$1, $$2}'

tidy: ## go mod tidy
	go mod tidy

build: ## Build local binary
	go build -o bin/rackwire ./cmd/rackwire

run: build ## Run locally on :$(PORT)
	ADDR=:$(PORT) DATA_PATH=data/rack.json ./bin/rackwire

up: ## Build and start via Compose
	RACKWIRE_PORT=$(PORT) $(COMPOSE) up -d --build
	@echo "rackwire at http://localhost:$(PORT)"

down: ## Stop Compose stack
	$(COMPOSE) down

restart: down up ## Restart

logs: ## Follow logs
	$(COMPOSE) logs -f

login: ## Login to GHCR
	@if [ -n "$$GH_TOKEN" ]; then \
		echo "$$GH_TOKEN" | docker login ghcr.io -u spakarl --password-stdin; \
	else \
		gh auth token | docker login ghcr.io -u spakarl --password-stdin; \
	fi

push: ## Build image and push to GHCR
	docker build -t $(IMAGE):$(TAG) .
	docker push $(IMAGE):$(TAG)

publish: login push ## Login + push

clean: ## Remove containers/volumes/image
	$(COMPOSE) down --rmi local --volumes --remove-orphans || true
	docker rmi $(IMAGE):$(TAG) 2>/dev/null || true
	rm -rf bin
