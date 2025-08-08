# 🛡️ Kubernetes CVE Scanner with Custom Controller + Admission Webhook

This project includes a **Kubernetes custom controller** that:
- Automatically creates **Services** and **Ingresses** for every `Deployment`.
- Integrates with a **Validating Admission Webhook** to scan container images using **Trivy**.
- Optionally allows skipping CVE checks with an environment variable.

---

## 🚀 Installation Guide

### 1️⃣ Create a Kubernetes Cluster

Make sure you have a running Kubernetes cluster (like KinD, Minikube, or EKS).

---

### 2️⃣ Install `cert-manager`

```bash
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.18.2/cert-manager.yaml
```
This will install the necessary CRDs and controllers for certificate management.

### 3️⃣ Deploy Trivy as a Service
```bash
kubectl apply -f docs/trivy-manifest/deployment.yml
kubectl apply -f docs/trivy-manifest/service.yml
```
Trivy will act as the backend scanner for your webhook.

### 4️⃣ Create Cluster Role & Bindings
* Grant required permissions for:
    - Deployments
    - Services 
    - Secrets 
    - Ingresses 
    - ValidatingWebhookConfigurations
```bash
kubectl apply -f manifest/cluster-permission.yaml

```

### 5️⃣ Deploy Controller + Webhook
* This manifest includes:
    - Namespace
    - Deployment
    - Service
    - TLS Issuers + Certs
    - ValidatingWebhookConfiguration

```bash
kubectl apply -f manifest/k8s-controller-webhook.yaml
```
### 6️⃣ Test Webhook
```bash
# contain cve
kubectl apply -f manifest/webhook-example/initContainerDeployment.yml 
# look for first time it might fail (look at the logs of the application (k8s-custom-controller) and 
# see if they return a long list of CVE -> then start creating again (Working on to optimize) 

# pure zero cve (does not contain cve) 
kubectl apply -f manifest/webhook-example/pureZeroCVE.yml

# contain cve but bypass (i mean create the deployment even after having CVE) 
# due to this parameter `name: BYPASS_CVE_DENIED` set as yes or true
kubectl apply -f manifest/webhook-example/ZeroInitCVE.yml
```
### Todo: 
- Better docs and guide

Happy Scan-ing!