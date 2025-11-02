APP_NAME ?= resilnce
WORKDIR  ?= apps/server
PORT     ?= 8080

# ---- Cloud settings ----
PROJECT_ID ?= resilnce
REGION     ?= europe-north1
SERVICE    ?= $(APP_NAME)

# ---- Image names ----
IMAGE_LOCAL  ?= $(APP_NAME):dev
IMAGE_REMOTE ?= gcr.io/$(PROJECT_ID)/$(APP_NAME)

# Container name for local runs
CONTAINER ?= $(APP_NAME)-dev

export GOOGLE_CLIENT_ID
export GOOGLE_CLIENT_SECRET
export FRONTEND_BASE_URL
export SIGNED_COOKIE_SECRET

.PHONY: help dev server docker-build docker-run stop logs curl tidy clean \
        docker-tag docker-push gcloud-auth deploy release

help:
	@echo "Targets:"
	@echo "  make dev              - run locally with 'go run' (PORT=$(PORT))"
	@echo "  make server           - build & run Docker image locally (PORT=$(PORT))"
	@echo "  make docker-build     - build image for linux/amd64 from $(WORKDIR)"
	@echo "  make docker-tag       - tag local image -> $(IMAGE_REMOTE)"
	@echo "  make docker-push      - push tagged image to GCR"
	@echo "  make deploy           - deploy to Cloud Run ($(REGION))"
	@echo "  make release          - build + tag + push + deploy"
	@echo "  make stop             - stop local container"
	@echo "  make logs             - tail logs from the local container"
	@echo "  make curl             - quick health check curl"
	@echo "  make tidy             - go mod tidy in $(WORKDIR)"
	@echo "  make clean            - remove local dev image"

dev: tidy
	cd $(WORKDIR) && PORT=$(PORT) go run .

tidy:
	cd $(WORKDIR) && go mod tidy

server: docker-run

# Build the Docker image from the root, using the Dockerfile in $(WORKDIR)
docker-build:
	docker build --platform linux/amd64 -t $(IMAGE_LOCAL) $(WORKDIR)

docker-run: docker-build
	docker run --rm \
		--env-file $(WORKDIR)/.env \
		-p $(PORT):$(PORT) \
		--name $(CONTAINER) \
		$(IMAGE_LOCAL)

stop:
	-@docker stop $(CONTAINER) >/dev/null 2>&1 || true

logs:
	docker logs -f $(CONTAINER)

curl:
	curl -i http://localhost:$(PORT)/resilnce

clean:
	-@docker rmi $(IMAGE_LOCAL) >/dev/null 2>&1 || true

# ---- Cloud bits ----

# One-time (or rarely) to let Docker use gcloud creds for gcr.io
gcloud-auth:
	gcloud auth configure-docker gcr.io

docker-tag:
	docker tag $(IMAGE_LOCAL) $(IMAGE_REMOTE)

docker-push: docker-tag
	docker push $(IMAGE_REMOTE)

# Reads FRONTEND_BASE_URL and SIGNED_COOKIE_SECRET from your shell env.
# Example:
#   FRONTEND_BASE_URL=http://localhost:5173 SIGNED_COOKIE_SECRET=xyz make deploy
deploy:
	gcloud run deploy $(SERVICE) \
	  --image $(IMAGE_REMOTE) \
	  --region $(REGION) \
	  --allow-unauthenticated \
	  --set-env-vars "FRONTEND_BASE_URL=$$FRONTEND_BASE_URL,SIGNED_COOKIE_SECRET=$$SIGNED_COOKIE_SECRET,GOOGLE_CLIENT_ID=$$GOOGLE_CLIENT_ID,GOOGLE_CLIENT_SECRET=$$GOOGLE_CLIENT_SECRET"

# One-shot: build -> push -> deploy
release: docker-build docker-push deploy

#
