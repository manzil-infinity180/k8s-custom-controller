APP_NAME = k8s-custom-controller
DOCKER_USER = manzilrahul
VERSION ?= 1.0.9
IMAGE_NAME = $(DOCKER_USER)/$(APP_NAME)

# 🖼️ Logo banner
define ascii_banner
	@cat banner.txt
endef

print-vars:
	$(call ascii_banner)
	@echo "🔧 APP_NAME     = $(APP_NAME)"
	@echo "👤 DOCKER_USER  = $(DOCKER_USER)"
	@echo "🏷️  VERSION      = $(VERSION)"
	@echo "📦 IMAGE_NAME   = $(IMAGE_NAME)"

run:
	$(call ascii_banner)
	@echo "▶️  Running Go app locally..."
	go run main.go

build:
	$(call ascii_banner)
	@echo "🛠️  Building Go binary..."
	go build -o $(APP_NAME)

fmt:
	$(call ascii_banner)
	@echo "🧹 Formatting Go code..."
	go fmt ./...

up:
	$(call ascii_banner)
	@echo "📦 Starting Docker Compose..."
	docker-compose up --build

down:
	$(call ascii_banner)
	@echo "🛑 Stopping Docker Compose..."
	docker-compose down

build-image:
	$(call ascii_banner)
	@echo "🐳 Building Docker image $(IMAGE_NAME):$(VERSION) and :latest..."
	docker build -t $(IMAGE_NAME):$(VERSION) -t $(IMAGE_NAME):latest .

push-image:
	$(call ascii_banner)
	@echo "📤 Pushing Docker image..."
	docker push $(IMAGE_NAME):$(VERSION)
	docker push $(IMAGE_NAME):latest
