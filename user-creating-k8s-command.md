### User creation in Kubernetes

Generate new ssl key:
--------------------
```bat
 openssl genrsa -out asimehsan.key 2048
```

Generate new certificate signing request for the issuer authority:
------------------------------------------------------------------
```rs
#CN is the name of the user
openssl req -new -key asimehsan.key -out asimehsan.csr -subj "/CN=asimehsan"

OR

#O is the group name. When you will create the rolebinding do the binding based on group name.
openssl req -new -key asimehsan.key -out asimehsan.csr -subj "/CN=asimehsan/O=cluster:manager"
```

Create manifest file csr_template.yaml:
---------------------------------------
```go
cat <<EOF > csr_template.yaml
apiVersion: certificates.k8s.io/v1
kind: CertificateSigningRequest
metadata:
name: asimehsan-csr
spec:
request: <Base64_encoded_CSR>
signerName: kubernetes.io/kube-apiserver-client
usages:
- client auth
  EOF
```

Save the certificate signing request in base64 encoded in variable CSR_CONTENT:
-------------------------------------------------------------------------------
```rs
CSR_CONTENT=$(cat asimehsan.csr | base64 | tr -d '\n')
```

Put the encoded certificate signing request in template manifest:
-----------------------------------------------------------------
```rs
sed "s|<Base64_encoded_CSR>|$CSR_CONTENT|" csr_template.yaml > asimehsan_csr.yaml
```

Create the csr resource:
----------------------- 
```rs
kubectl create -f asimehsan_csr.yaml
kubectl get csr
```

Do approval as cluster admin user:
---------------------------------
```rs
kubectl certificate approve asimehsan-csr
```

Fetch the issued certificate:
-----------------------------
```rs
kubectl get csr asimehsan-csr -o jsonpath='{.status.certificate}' | base64 --decode > asimehsan.crt
```

Take a look on current kubeconfig used:
-------------------------------------
```rs
kubectl config view
```

Take a look on the ssl certs directory:
--------------------------------------
```rs
ls /etc/kubernetes/pki/
```

Generate new kubeconfig file:
-----------------------------
```rs
# Set Cluster Configuration:
kubectl config set-cluster kubernetes --server=https://<API-Server-IP>:6443 --certificate-authority=/etc/kubernetes/pki/ca.crt --embed-certs=true --kubeconfig=asimehsan.kubeconfig

# Set Credentials for asimehsan:
kubectl config set-credentials asimehsan --client-certificate=asimehsan.crt --client-key=asimehsan.key --embed-certs=true --kubeconfig=asimehsan.kubeconfig

# Set asimehsan Context:
kubectl config set-context asimehsan-context --cluster=kubernetes --namespace=default --user=asimehsan --kubeconfig=asimehsan.kubeconfig

# Use asimehsan Context:
kubectl config use-context asimehsan-context --kubeconfig=asimehsan.kubeconfig


# Set KUBECONFIG environment variable pointing to asimehsan.kubeconfig
export KUBECONFIG=<path>/asimehsan.kubeconfig

# Validate the user rights from admin user
kubectl auth can-i list pods --as system:serviceaccount:dev:user1 -n dev
kubectl auth can-i list pods --as asimehsan -n dev

# Validate by user directly
kubectl auth can-i list pods -n dev
```


Reference 
---

-> https://github.com/asimehsan/devops-vu/blob/main/Install%20k8s%20locally/RBAC%20User%20.txt \
-> https://youtu.be/w0X4h_etgxA?si=OJDhY_-2ApIo3d3t