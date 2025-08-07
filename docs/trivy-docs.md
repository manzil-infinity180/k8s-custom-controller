```bash
kubectl exec -it <your-controller-pod> -- nslookup trivy-server-service.default.svc
kubectl exec -it <your-controller-pod> -- curl http://trivy-server-service.default.svc:8080/healthz

---
kubectl exec -it k8s-custom-controller-5c7d47fdb7-69757 -- curl http://trivy-server-service.default.svc:8080/healthz
ok
---
k exec -it k8s-custom-controller-5c7d47fdb7-69757 -- bash
k8s-custom-controller-5c7d47fdb7-69757:/etc# cat resolv.conf
search example1.svc.cluster.local svc.cluster.local cluster.local
nameserver 10.96.0.10
options ndots:5
```

