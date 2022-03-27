# Interacting with Kubernetes Workloads
In this example, we'll explore how to interact with the Kubernetes API using a Go client to create a namespace, ingress, service, and deployment to run an Nginx hello world service.

The service runs locally in KinD streaming logs from the pods in the service to STDOUT. After the service is started, open a browser to http://localhost:8080/hello. You should see the request stream to STDOUT, and you should be greeted with a page describing the request and the server that served it. If you refresh the page, you should see the server name change indicating the requests are load balancing across the 2 pod replicas in the deployment.

## Required tools
- docker
- kubectl
- kind

## Running the code
```shell
kind create cluster --name workloads --config kind-config.yaml
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/main/deploy/static/provider/kind/deploy.yaml
kubectl wait --namespace ingress-nginx \
  --for=condition=ready pod \
  --selector=app.kubernetes.io/component=controller \
  --timeout=90s
go run .
```

## Deleting the KinD cluster
Deleting the KinD cluster will clean up all resources used for this example.
```shell
kind delete cluster --name workloads
```
