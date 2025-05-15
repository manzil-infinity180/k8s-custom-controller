```go
docker run --rm --name k8s-custom-controller \
  --network host \
  -v $HOME/.kube:/root/.kube \
  k8s-custom-controller-controller:latest

```
