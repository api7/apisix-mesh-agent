# ETCD v3 API Mimicking

This article explains the mimicking of [ETCD v3 API](https://etcd.io/docs/current/learning/api/).

## Table of Contents

- [Why](#why)
- [Mimicking Principles](#mimicking-principles)
- [Data Source](#data-source)
- [Key Value Metadata](#key-value-metadata)

## Why

The [Apache APISIX](http://apisix.apache.org/) is coupled with [ETCD v3](https://etcd.io/), and it's not feasible
to let it support other data centers in a short period. The mimicking of ETCD v3 API thereby comes in.

## Mimicking Principles

All APIs used by Apache APISIX should be supported well.
So far Apache APISIX uses the following APIs:

- `RangeRequest`, using it to fetch configurations totally;
- `WatchRequest`, using it to watch the newest configuration changes;
- `PutRequest`, using it to uploading some data;

The `RangeRequest` and `WatchRequest` will be implemented meticulously;
As for the `PutRequest`, since the data uploaded from Apache APISIX are not used by apisix-mesh-agent,
so the `PutRequest` can be dummy. Just lie to Apache APISIX for avoiding throwing too many error logs.

For the sake of observability, the mimic ETCD API should be accessed normally from [etcdctl](https://etcd.io/docs/current/dev-guide/interacting_v3/),
so [gRPC](https://grpc.io/) will be chosen as the protocol. However, Apache APISIX relies on HTTP restful APIs, which supported by
the [gRPC-Gateway](https://grpc-ecosystem.github.io/grpc-gateway/), so it's also required for apisix-mesh-agent.

## Data Source

Although apisix-mesh-agent mimics the ETCD v3 API, it doesn't inherit the persistence of ETCD.

Data come from the in memory cache inside apisix-mesh-agent, For `RangeRequest`, just iterating
the cache and output them; For `WatchRequest`, each time new events arrived from [Privisioner](./the-internal-of-apisix-mesh-agent.md#Provisioner), it also
deliveries them to the watch clients.

## Key Value Metadata

[Metadata](https://github.com/etcd-io/etcd/blob/master/api/mvccpb/kv.proto#L12) in the KV response of
ETCD v3 API. All of them should also be filled but not be accurate, as long as values like `mod_revision` doesn't drift.
