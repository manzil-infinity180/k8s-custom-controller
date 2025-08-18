<div align="center">
<p align="center">
<img width="960" height="309" alt="final-k8s" src="https://github.com/user-attachments/assets/e5ef535e-a07a-4cd5-9fbd-926a0c62cf39" />
</p>

[![Watch the demo video](https://github.com/user-attachments/assets/4ba51960-d9d2-4ac4-9272-c4ee3c5cf262)](https://www.youtube.com/watch?v=mAr62XBVbmg)
</div>

> 📽️ Click the image above to watch the full 25-minute walkthrough on YouTube.  
> It includes setup, explanation, CVE scan demo, and auto resource creation.


# 🛡️ Kubernetes CVE Scanner with Custom Controller + Admission Webhook

This project includes a **Kubernetes custom controller** that:
- Automatically creates **Services** and **Ingresses** for every `Deployment`.
- Integrates with a **Validating Admission Webhook** to scan container images using **Trivy**.
- Optionally allows skipping CVE checks with an environment variable.

---

## 🚀 Installation Guide

### 1️⃣ Create a Kubernetes Cluster

Make sure you have a running Kubernetes cluster (like KinD, Minikube, or EKS).


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
> Note: We are running using trivy client you can see the command [here](https://github.com/aquasecurity/trivy/discussions/2119)

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

```ts
kubectl apply -f manifest/k8s-controller-webhook.yaml
```
### 6️⃣ Test Webhook
```bash
# contain cve
$ kubectl apply -f manifest/webhook-example/initContainerDeployment.yml
# look for first time it might fail (look at the logs of the application (k8s-custom-controller) and 
# see if they return a long list of CVE -> then start creating again (Working on to optimize) 

# pure zero cve (does not contain cve) 
$ kubectl apply -f manifest/webhook-example/pureZeroCVE.yml

# contain cve but bypass (i mean create the deployment even after having CVE) 
# due to this parameter `name: BYPASS_CVE_DENIED` set as yes or true
$ kubectl apply -f manifest/webhook-example/ZeroInitCVE.yml
```
> Todo:
> Better docs and guide

<p align="center">
<img width="450" height="450" alt="image" src="https://github.com/user-attachments/assets/92fe17a5-bffe-469d-beb3-0769bb85d4a5" />
</p>

## Author

Built with 💙 by **Rahul Vishwakarma**

> Happy Scan-ing!
