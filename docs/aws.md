```
eksctl create cluster \
    --name demo-cluster \
    --region us-east-1 \
    --node-type t2.micro \
    --nodes 2 \
    --managed
```

```bash
eksctl create cluster \
  --name demo-cluster \
  --region us-east-1 \
  --node-type t3.medium \
  --nodes 2 \
  --managed

```