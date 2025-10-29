APP_NAME ?= apps-server
WORKDIR  ?= apps/server
PORT     ?= 8080

IMAGE     ?= $(APP_NAME):dev
CONTAINER ?= $(APP_NAME)-dev

.PHONY: help dev server docker-build docker-run stop logs curl tidy clean

help:
	@echo "Targets:"
	@echo "  make dev            - run locally with 'go run' (PORT=$(PORT))"
	@echo "  make server         - build & run Docker image (PORT=$(PORT))"
	@echo "  make stop           - stop the running Docker container"
	@echo "  make logs           - tail logs from the Docker container"
	@echo "  make curl           - quick health check curl"
	@echo "  make tidy           - go mod tidy in $(WORKDIR)"
	@echo "  make clean          - remove local dev image"

dev: tidy
	cd $(WORKDIR) && PORT=$(PORT) go run .

tidy:
	cd $(WORKDIR) && go mod tidy

server: docker-run

docker-build:
	docker build -t $(IMAGE) $(WORKDIR)

docker-run: docker-build
	docker run --rm \
		-p $(PORT):$(PORT) \
		-e PORT=$(PORT) \
		--name $(CONTAINER) \
		$(IMAGE)

stop:
	-@docker stop $(CONTAINER) >/dev/null 2>&1 || true

logs:
	docker logs -f $(CONTAINER)

curl:
	curl -i http://localhost:$(PORT)/resilnce

clean:
	-@docker rmi $(IMAGE) >/dev/null 2>&1 || true

cookie: 
	curl -i --cookie "sess=IdP_info" http://localhost:$(PORT)/cookie

# make get CKI=""
get:
	curl -i --cookie "sess=$(CKI)" http://localhost:$(PORT)/get


	
