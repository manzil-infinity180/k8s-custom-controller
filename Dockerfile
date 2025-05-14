# Build stage
FROM golang:1.24 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
ARG TARGETARCH=amd64
#RUN go build -o k8s-custom-controller
RUN CGO_ENABLED=0 GOOS=linux GOARCH=$TARGETARCH go build -o backend main.go


# Final stage
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/backend .

# Allow access to Kubernetes API via a volume mount for kubeconfig
VOLUME ["/root/.kube"]
EXPOSE 8000
CMD ["./backend"]
