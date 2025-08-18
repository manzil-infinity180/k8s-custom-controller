## Create EKS Cluster
```bash
eksctl create cluster \
  --name demo-cluster \
  --region us-east-1 \
  --node-type t3.medium \
  --nodes 2 \
  --managed

```

## Install cert-manager inside your cluster
```bash
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.18.2/cert-manager.yaml
```

## Install helm chart 
Visit: https://manzil-infinity180.github.io/k8s-custom-controller/

```bash
helm repo add k8s-custom-controller https://manzil-infinity180.github.io/k8s-custom-controller

helm repo update

helm install my-release k8s-custom-controller/deploydefender # --version 0.1.3
```

Or other way you can do with is by installation manually
Check the `Readme.md` or `bash script`

## Experiment with the example manifest files
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

## Delete your eks cluster
```bash
# you are done with all practice 
eksctl delete cluster --name demo-cluster --region us-east-1
```