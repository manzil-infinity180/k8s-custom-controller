# =========================
# Build Stage
# =========================
FROM golang:1.24 AS builder

WORKDIR /app

# Download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code
COPY . .
ARG TARGETARCH=amd64

# Build the Go binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=$TARGETARCH go build -o backend main.go


# =========================
# Final Stage with Trivy
# =========================
FROM alpine:3.20

# Install required packages
RUN apk --no-cache add \
    ca-certificates \
    curl \
    bash \
    tar \
    gzip \
    libc6-compat

# Install Trivy (static binary)
ENV TRIVY_VERSION=0.55.2
RUN curl -sfL https://github.com/aquasecurity/trivy/releases/download/v${TRIVY_VERSION}/trivy_${TRIVY_VERSION}_Linux-64bit.tar.gz \
    | tar zx -C /usr/local/bin/ trivy

# Verify installation
RUN trivy --version

WORKDIR /root/

# Copy the Go binary
COPY --from=builder /app/backend .

# Allow access to Kubernetes API via a volume mount for kubeconfig
VOLUME ["/root/.kube"]

# Expose webhook port
EXPOSE 8000

CMD ["./backend"]
