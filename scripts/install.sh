#!/usr/bin/env bash

set -euo pipefail

SCRIPT_NAME="install-controllerx.sh"
k8s_platform="kind"
cluster="controllerx"
cluster_log_dir=$(mktemp -d)
use_helm=false

# Parse command-line arguments
while [[ "$#" -gt 0 ]]; do
    case $1 in
        --platform) k8s_platform="$2"; shift ;;
        --helm) use_helm=true ;;
        -X) set -x ;;
        -h|--help)
            echo "Usage: ${SCRIPT_NAME} [--platform <kind|k3d>] [--helm] [-X] [-h|--help]" >&2
            echo "  --platform <kind|k3d>   Choose Kubernetes platform (default: kind)"
            echo "  --helm                  Install via Helm instead of manual manifests"
            echo "  -X                      Enable debug mode (set -x)"
            echo "  -h, --help              Show this help message"
            exit 0
            ;;
        *)
            echo "Unknown parameter passed: $1" >&2
            exit 1
            ;;
    esac
    shift
done

if [[ "$k8s_platform" != "kind" && "$k8s_platform" != "k3d" ]]; then
    echo "Invalid platform specified: $k8s_platform"
    echo "Supported platforms are: kind, k3d"
    exit 1
fi

echo "Selected Kubernetes platform: $k8s_platform"
echo "Helm mode: $use_helm"

# Cleanup old cluster + context
if kubectl config get-contexts -o name | grep -q "^${cluster}$"; then
    echo "🧹 Removing old context $cluster"
    kubectl config delete-context "$cluster" || true

    if [ "$k8s_platform" == "kind" ]; then
        echo "🧹 Deleting old KinD cluster $cluster"
        kind delete cluster --name "$cluster" || true
    elif [ "$k8s_platform" == "k3d" ]; then
        echo "🧹 Deleting old k3d cluster $cluster"
        k3d cluster delete "$cluster" || true
    fi
fi

echo -e "\033[33m✔\033[0m Cleanup completed"

# Create cluster
if [ "$k8s_platform" == "kind" ]; then
    echo "🔧 Creating KinD cluster: $cluster"
    kind create cluster --name "$cluster" &>"${cluster_log_dir}/${cluster}.log"
else
    echo "🔧 Creating K3d cluster: $cluster"
    k3d cluster create "$cluster" &>"${cluster_log_dir}/${cluster}.log"
fi

echo "✔ Cluster $cluster created"
kubectl config rename-context "${k8s_platform}-${cluster}" "${cluster}" >/dev/null 2>&1
kubectl config use-context "${cluster}"

# Install cert-manager
echo "📦 Installing cert-manager..."
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.18.2/cert-manager.yaml
kubectl rollout status deployment -n cert-manager cert-manager --timeout=120s || true

if [ "$use_helm" = true ]; then
    echo "🚀 Installing Controller via Helm..."
    helm repo add k8s-custom-controller https://manzil-infinity180.github.io/k8s-custom-controller
    helm repo update
    helm install my-release k8s-custom-controller/deploydefender # --version 0.1.3
    echo -e "\n✅ Helm installation completed!"
    echo "👉 Docs: ./docs/aws.md"
else
  # Deploy Trivy (manual mode)
  echo "📦 Deploying Trivy..."
  kubectl apply -f docs/trivy-manifest/deployment.yml
  kubectl apply -f docs/trivy-manifest/service.yml
  kubectl rollout status deployment trivy -n default --timeout=120s || true

  # Apply RBAC
  echo "🔑 Applying Cluster Roles..."
  kubectl apply -f manifest/cluster-permission.yaml

  # Deploy Controller + Webhook
  echo "🚀 Deploying Controller + Webhook..."
  kubectl apply -f manifest/k8s-controller-webhook.yaml

fi

echo -e "\n✅ Manual setup completed!"
echo -e "👉 Try the test manifests:\n"
cat <<EOF
# Contains CVEs
kubectl apply -f manifest/webhook-example/initContainerDeployment.yml

# Zero CVE image
kubectl apply -f manifest/webhook-example/pureZeroCVE.yml

# Contains CVEs but bypass allowed
kubectl apply -f manifest/webhook-example/ZeroInitCVE.yml
EOF
