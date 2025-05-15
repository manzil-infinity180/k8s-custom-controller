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
make build-image         # builds 1.0.1 and latest
make push-image          # pushes 1.0.1 and latest

# Override version if needed:
make build-image VERSION=1.0.2
make push-image VERSION=1.0.2

make build-image APP_NAME=custom-controller DOCKER_USER=yourusername VERSION=2.0.0
```