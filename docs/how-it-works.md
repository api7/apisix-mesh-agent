# How It Works

This article explains how apisix-mesh-agent extends [Apache APISIX](https://apisix.apache.org) as the Service Mesh sidecar.

## Run Mode

apisix-mesh-agent can be run alone or bundled with Apache APISIX.
It depends on how you pass the running options to it.

If you want to run it alone, for instance, you want to run the apisix-mesh-agent and APISIX in different
Pods/VMs so that the apisix-mesh-agent can be shared by multiple APISIX instances, then just pass `--run-mode standalone`
for it. In such a case, the `etcd.host` configuration in APISIX should be configured to the gRPC listen address
of apisix-mesh-agent.

```shell
/path/to/apisix-mesh-agent sidecar --provisioner xds-v3-file --xds-watch-files /path/to/xds-assets --run-mode standalone
```

As a common pattern, sidecar and apps are always deployed together, if you run apisix-mesh-agent under the "bundle" mode, it
will launch the Apache APISIX and close it when you shut apisix-mesh-agent down.

```shell
/path/to/apisix-mesh-agent sidecar --apisix-bin-path /path/to/bin/apisix --apisix-home-path /path/to/apisix/ --provisioner xds-v3-file --xds-watch-files /path/to/xds-assets --run-mode bundle
```

You should pass the correct Apache APISIX binary path and home path, apisix-mesh-agent render [a configuration file](../pkg/sidecar/apisix/config.yaml) for it, each time you start apisix-mesh-agent,
configuration file will be written to `/path/to/apisix/conf/config-default.yaml`.
