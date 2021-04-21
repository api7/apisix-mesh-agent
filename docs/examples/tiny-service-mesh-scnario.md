Tiny Mesh Scanario
===================

This post builds a tiny service mesh and tests it.

```shell
                                          +----------------------------+                                             
                                          |                            |                                             
                                          |                            |                                             
            +----------------------------->                            |                                             
            |                             |           istio            |<--------+                                   
            |                             |                            |         |                                   
            |                             |                            |         |                                   
            |                             +----------------------------+         |                                   
            |                                                                    |                                   
            |                                                                    |                                   
            |                                                                    |                                   
            |                                                                    |                                   
+-----------|------------------------------------+                  +------------|----------------------------------+
|           |                                    |                  |            |                                  |
|           |                                    |                  |            |                                  |
|+----------|-------+  2    +------------------+ |       3          | +----------|-------+ 4   +------------------+ |
||                  ------->|                  |--------------------->|                  |---->|                  | |
||apisix-mesh-agent |       |nginx             | |                  | |apisix-mesh-agent |     |   httpbin        | |
||                  <-------|                  |<---------------------|                  |<----|                  | |
|+-----------^---|--+  7    +------------------+ |       6          | +------------------+  5  +------------------+ |
|            |   |                               |                  |                                               |
+------------|---|-------------------------------+                  +-----------------------------------------------+
             |   |                                                                                                   
             |   |                                                                                                   
             |   |                                                                                                   
             |   |                                                                                                  
             |   |                                                                                                   
           1 |   |8                                                                                                  
             |   |                                                                                                   
             |   |                                                                                                   
             |   |                                                                                                   
             |   |                                                                                                   
             |   |                                                                                                   
             |   |                                                                                                   
             |   |                                                                                                   
             |   v                                                                                                                                                                                           
```

HTTP Requests will be sent to the nginx pod and redirected to httpbin, both nginx and httpbin pods have the apisix-mesh-agent sidecar, and requests will be intercepted by it.

Prequisites
-----------

First you should install APISIX Mesh as per the [guide](../istio-mesh.md).

Then creating the namespace that used to deploy nginx and httpbin.

```shell
kubectl create namespace app
kubectl label namespace app istio-injection=enabled
```

The above commands also mark the namespace `app` as injectable, so that pods created there will be injected by [Istio](https://istio.io).

Deployment
----------

```shell
kubectl run httpbin --image kennethreitz/httpbin --port 80 -n app
kubectl expose pod/httpbin --port 80 -n app

kubectl apply -n app -f - << EOS
apiVersion: v1
kind: ConfigMap
metadata:
  name: nginx-conf
data:
  httpbin.conf: |
    server {
        listen 80 reuseport;
        location / {
                proxy_http_version 1.1;
                proxy_set_header Connection "";
                proxy_pass http://httpbin;
                proxy_set_header Host httpbin;
        }
    }
---
apiVersion: v1
kind: Pod
metadata:
  name: nginx
  labels:
    app: nginx
spec:
  volumes:
  - name: conf
    configMap:
      name: nginx-conf
  containers:
  - name: nginx
    image: nginx
    imagePullPolicy: IfNotPresent
    ports:
    - containerPort: 80
      protocol: TCP
      name: http
    volumeMounts:
    - name: conf
      mountPath: /etc/nginx/conf.d
---
apiVersion: v1
kind: Service
metadata:
  name: nginx
spec:
  selector:
    app: nginx
  ports:
  - name: http
    targetPort: 80
    port: 80
    protocol: TCP
EOS

kubectl run box --image=centos:7 --command sleep 36000
```

The above commands create two pods (nginx, http), and expose them respectively. The nginx pod proxies requests to the httpbin service.

The last command creates a pod which used as the http client. We'll send requests from there.

Test
----

```
kubectl exec box -- curl http://nginx.app.svc.cluster.local/get -H 'Host: httpbin' -s -v
* About to connect() to nginx.app.svc.cluster.local port 80 (#0)
*   Trying 10.96.190.178...
* Connected to nginx.app.svc.cluster.local (10.96.190.178) port 80 (#0)
> GET /get HTTP/1.1
> User-Agent: curl/7.29.0
> Accept: */*
> Host: httpbin
>
< HTTP/1.1 200 OK
< Server: openresty
< Date: Wed, 21 Apr 2021 09:52:32 GMT
< Content-Type: application/json
< Content-Length: 212
< Connection: keep-alive
< Access-Control-Allow-Origin: *
< Access-Control-Allow-Credentials: true
< Via: APISIX
< Via: APISIX
<
{ [data not shown]
* Connection #0 to host nginx.app.svc.cluster.local left intact
{
  "args": {},
  "headers": {
    "Accept": "*/*",
    "Host": "httpbin",
    "User-Agent": "curl/7.29.0",
    "X-Forwarded-Host": "httpbin"
  },
  "origin": "10.244.3.26",
  "url": "http://httpbin/get"
}
```

The `Via` header is added by each inbound APISIX proxy, as the request goes through two sidecars, so there are two `Via` headers. The respoonse body is expected which returns request information by httpbin.

Uninstall
---------

```shell
kubectl delete namespace app
kubectl expose pod/httpbin --port 80 -n app


```