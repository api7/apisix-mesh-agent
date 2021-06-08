# Charts

## Nacos

Add `livenessProbe`:

```yaml
  readinessProbe:
    httpGet:
      path: /nacos
      port: {{ .Values.nacos.serverPort }}
    failureThreshold: 3
    initialDelaySeconds: 10
    periodSeconds: 8
    successThreshold: 1
    timeoutSeconds: 1
```

## Istio Discovery Nacos

Change `injection-template`:

```yaml
    args:
    - --provisioner
    - nacos
    - --nacos-source
    - "http://nacos-cs.nacos-e2e.svc.{{ .Values.global.proxy.clusterDomain }}:8848"
```
