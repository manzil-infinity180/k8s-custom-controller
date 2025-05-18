```go
docker run --rm --name k8s-custom-controller \
  --network host \
  -v $HOME/.kube:/root/.kube \
  k8s-custom-controller-controller:latest

```
```go
kubectl create deployment my-deployment --image=nginx:latest --labels=app=nginx,env=prod
```
```go
// https://patorjk.com/software/taag/#p=display&h=1&v=0&f=Slant&t=k8s%20Controller%0A 
// for creating the ASCII Banner
make build-image         # builds 1.0.1 and latest
make push-image          # pushes 1.0.1 and latest

# Override version if needed:
make build-image VERSION=1.0.2
make push-image VERSION=1.0.2

make build-image APP_NAME=custom-controller DOCKER_USER=yourusername VERSION=2.0.0
```
```md
# Helm
https://jhooq.com/building-first-helm-chart-with-spring-boot/
https://jhooq.com/convert-kubernetes-yaml-into-helm/
```

```md
# how we will port forward svc
> kubectl port-forward svc/k8s-custom-controller-service 8080:80
```

```md
# this is another way to build depend on platform

docker buildx build --platform linux/amd64 -t manzilrahul/k8s-custom-controller:1.0.3 .
```

```md
kubectl get nodes -o wide
```
