## Kubeconfig

kubectl uses one file which is kubeconfig to authenticate itself 

if client want then it need private key and client certificate 

## How to access the kubeconfig file on our machine 

```bash
rahulxf@Rahuls-MacBook-Air-3 ~ % cd $HOME/.kube/  

rahulxf@Rahuls-MacBook-Air-3 .kube % ls -l
total 160
drwxr-x---@ 4 rahulxf  staff    128 Jan 17 22:46 cache
-rw-------@ 1 rahulxf  staff  31948 Feb 17 10:48 config
-rw-r--r--  1 rahulxf  staff   7973 Feb  4 16:01 karmada-apiserver.config
-rw-------  1 rahulxf  staff  13415 Feb 11 19:28 karmada.config
-rw-r--r--  1 rahulxf  staff      4 Feb 17 10:43 kubectx
drwxr-xr-x  4 rahulxf  staff    128 Feb 17 10:48 kubens
-rw-r--r--  1 rahulxf  staff  16652 Feb  2 22:01 members.config

rahulxf@Rahuls-MacBook-Air-3 .kube %

rahulxf@Rahuls-MacBook-Air-3 .kube % vim config
rahulxf@Rahuls-MacBook-Air-3 .kube % vim config
```

* config is the kubeconfig file 

```
apiVersion: v1
clusters:
- cluster:
certificate-authority-data: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSURQRENDQWlTZ0F3SUJBZ0lDQm5Zd0RRWUpLb1pJa......
server: https://cp1.localtest.me:9443

name: cp1-cluster
- cluster:
server: ""

name: its1
- cluster:
certificate-authority-data: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUJlRENDQVIyZ0F3SUJBZ0lCQURBS0JnZ3Foa2pPU....
server: https://its1.localtest.me:9443
name: its1-cluster

contexts:
- context:
cluster: kind-cluster1
user: kind-cluster1
name: cluster1

- context:
cluster: kind-cluster2
user: kind-cluster2
name: cluster2

users:
- name: cp1-admin
user:
client-certificate-data: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSURSVENDQWkyZ0F3SUJBZ0lDQm5vd0RRWUpLb1pJaHZjTkFRRUxCUUF3UHpFVE1CRUdBMVVFQ2hNS1MzVmkKWlhKdVpYUmxjekVUTUJFR0ExVUVDeE1LUVZCSklGTmxjblpsY2pFVE1CRUdBMVVFQXhNS2EzVmlaWEp1WlhSbApjekFlRncweU5UQXhNVGN4TmpJek5UUmFGdzB6TlRB

client-key-data: LS0tLS1CRUdJTiBSU0EgUFJJVkFURSBLRVktLS0tLQpNSUlFcGdJQkFBS0NBUUVBM1l0MmgzVHp6NHgzYnU2akhJWXVDZHVKbWpTejNrSWtYVFczNEFHN2ZtR2hENS9DCjZNOFdtZGd1clFjU0doQVIyOENSaUhKUHoxckU4

- name: its1-admin

user:

client-certificate-data: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUJrRENDQVRlZ0F3SUJBZ0lJVmloSTBuYzg0dDR3Q2dZS

client-key-data: LS0tLS1CRUdJTiBFQyBQUklWQVRFIEtFWS0tNNDkKQXdFSG9VUURRZ0FFa1ZXYWNmbmwyTlg0L1d6NCthVS9JVzVyU05lSVhGZW5ROT

- name: kind-cluster1

user:
client-certificate-data: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSURLVENDQWhHZ0F3SUJBZ0lJR0ZhWGh1RFU4c2t3RFFZSktvWklW9

client-key-data: LS0tLS
```

How to add users 

```
$ kubectl config set-credentials devuser --client-certificate du.crt --client-key du.key

User "devuser" set.
```

You also need to map it , i mean you need to add context for this user for the cluster 

```
$ kubectl config set-context --cluster kind-kind --user devuser

Context "devuser-kind" created
```

```
- context:
	cluster: kind-kind
	user: devuser
  name: devuser-kind
```

If you want to check which context we are using 

`$ kubectl config current-context`

or 
you can install `kubectx` 

* Flow of kubectl for looking kubeconfig file 
  1) --kubeconfig flag to kubectl command
  2) Then it will look for the environmental variable (env)
  3) after in the last it will look for the $HOME/.kube/config file 

Suppose you want - 2,3 kubeconfig file as one file then you can do this like you can specify the kubeconfig file with colon(:) separated 

```
$ export KUBECONFIG=~/.kube/config:~/.kube/karmada.config:~/.kube/karmada-apiserver.config
```

<img width="1120" alt="Screenshot 2025-02-18 at 11 21 48 PM" src="https://github.com/user-attachments/assets/d1d742bf-d308-42a8-bcf6-41b4d5c881da" />

To authenticate the user to kubernetes cluster we will do client certificate management 

docker ps  (get the id )
docker exec -it <id> bash
cd /etc/kubernetes/pki 
ls -l (you will see the key, csr and other files )


* So for creating private key and csr key you can run this command to generate

<img width="1179" alt="shapes at 25-02-20 00 13 41" src="https://github.com/user-attachments/assets/2aedf2d0-8b1b-4a23-9ce0-c815020ec5fa" />

Generate new ssl key:
-----

```
$ openssl genrsa -out rahulxf.key 2048
```

Generate new certificate signing request for the issuer authority:
----

```
#CN is the name of the user
#O is the group name. When you will create the rolebinding do the binding based on group name. 
$ openssl req -new -key rahulxf.key -out rahulxf.csr -subj "/CN=rahulxf/0=developers"
```

<img width="1157" alt="Screenshot 2025-02-19 at 12 46 29 AM" src="https://github.com/user-attachments/assets/c3cf391d-1a0a-4c4d-867f-da607225247f" />

<img width="986" alt="Screenshot 2025-02-19 at 12 56 19 AM" src="https://github.com/user-attachments/assets/850c5a52-7c21-44b0-89cc-1ce229dca0cd" />

The next step is to creating the user and setting up the context between the user and cluster in kubeconfig file 
------

```
# Adding user 
$ kubectl config set-credentials rahulxf --client-certificate rahulxf.crt --client-key rahulxf.key

# Creating context for the user and cluster
$ kubectl config set-context rahulxf-kind --user rahulxf --cluster kind-cluster2

```
<img width="937" alt="Screenshot 2025-02-20 at 12 17 04 AM" src="https://github.com/user-attachments/assets/6b653c3b-49e8-48f4-add9-9859023d9fe8" />

* Here you can look for the kubeconfig file 

<img width="1499" alt="Screenshot 2025-02-19 at 1 09 37 AM" src="https://github.com/user-attachments/assets/367eda1b-0ae6-4604-a2e7-62793769e42a" />

* see your context using command

```
$ kubectl config current-context
$ kubectl config get-contexts
$ kubectl config use-context <context_name>

# OR use kubectx
$ kubectx
```
<img width="1094" alt="Screenshot 2025-02-19 at 1 10 23 AM" src="https://github.com/user-attachments/assets/093537f4-76b7-4f11-890e-77781eefa5cd" />

* allow namespaces 
<img width="1310" alt="Screenshot 2025-02-19 at 1 17 41 AM" src="https://github.com/user-attachments/assets/b9e2ca7c-ac1c-4b7c-9384-4ec96a951f15" />


* allow pods 
<img width="1310" alt="Screenshot 2025-02-19 at 1 23 04 AM" src="https://github.com/user-attachments/assets/9967a9df-bac0-42d9-9b3a-d177e764b2b5" />
