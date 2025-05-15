APP_NAME = k8s-custom-controller
DOCKER_USER = manzilrahul
VERSION ?= 1.0.1
IMAGE_NAME = $(DOCKER_USER)/$(APP_NAME)

# Run locally with .env loaded
run:
	go run main.go

# Build the Go binary
build:
	go build -o ${APP_NAME} main.go

# Run Docker Compose
up:
	docker-compose up --build

# Stop containers
down:
	docker-compose down

#Build Docker image
build-image:
	docker build -t $(IMAGE_NAME):$(VERSION) -t $(IMAGE_NAME):latest .

# Push Docker image
push-image:
	docker push $(IMAGE_NAME):$(VERSION)
	docker push $(IMAGE_NAME):latest

# Format Go code
fmt:
	go fmt ./...
