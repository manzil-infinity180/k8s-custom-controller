
##  Create your first Helm Chart
// https://jhooq.com/getting-start-with-helm-chart/

```bash
helm create helloworld
ls -lart | grep helloworld

tree helloworld 
helloworld
├── charts
├── Chart.yaml
├── templates
│  ├── deployment.yaml
│  ├── _helpers.tpl
│  ├── hpa.yaml
│  ├── ingress.yaml
│  ├── NOTES.txt
│  ├── serviceaccount.yaml
│  ├── service.yaml
│  └── tests
│      └── test-connection.yaml
└── values.yaml

```

```bash
helm install <FIRST_ARGUMENT_RELEASE_NAME> <SECOND_ARGUMENT_CHART_NAME>

First argument - Release name that you pick
Second argument - Chart you want to install

helm install myhelloworld helloworld

NAME: myhellworld
LAST DEPLOYED: Sat Nov  7 21:48:08 2020
NAMESPACE: default
STATUS: deployed
REVISION: 1
NOTES:
1. Get the application URL by running these commands:
  export NODE_PORT=$(kubectl get --namespace default -o jsonpath="{.spec.ports[0].nodePort}" services myhellworld-helloworld)
  export NODE_IP=$(kubectl get nodes --namespace default -o jsonpath="{.items[0].status.addresses[0].address}")
  echo http://$NODE_IP:$NODE_PORT
```

```bash
helm list -a


```

