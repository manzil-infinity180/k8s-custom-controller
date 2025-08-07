```bash 
# create cluster 
# Install cert-manager 
# add trivy - k apply -f docs/trivy-manifest/deployment.yml and then same for svc
# k apply -f manifest/k8s-controller-webhook.yaml (it contain everything, cert, tls secrets)
# add cluster permission for list, watch, create, get 
# k apply -f manifest/cluster-permission.yaml
```