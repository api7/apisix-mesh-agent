# ETCD v3 API Mimicking

This article explains the mimicking of [ETCD v3 API](https://etcd.io/docs/current/learning/api/), it also registers the
level of API conformance.

Note this it not a generic implementation for ETCD v3 API, it's specific for [Apache APISIX](http://apisix.apache.org/).

## Table of Contents

- [Why](#why)
- [Mimicking Principles](#mimicking-principles)
- [Data Source](#data-source)
- [Key Value Metadata](#key-value-metadata)
- [API Conformance](#api-conformance)
  - [RangeRequest](#rangerequest)

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

Since this solution is exclusive for Apache APISIX, addressable key formats are also fixed, basically key should be like:

- `/apisix/routes/{id}`
- `/apisix/upstreams/{id}`

## Data Source

Although apisix-mesh-agent mimics the ETCD v3 API, it doesn't inherit the persistence of ETCD.

Data come from the in memory cache inside apisix-mesh-agent, For `RangeRequest`, just iterating
the cache and output them; For `WatchRequest`, each time new events arrived from [Privisioner](./the-internal-of-apisix-mesh-agent.md#Provisioner), it also
deliveries them to the watch clients.

## Key Value Metadata

[Metadata](https://github.com/etcd-io/etcd/blob/master/api/mvccpb/kv.proto#L12) in the KV response of
ETCD v3 API. apisix-mesh-agent mimics the revision, it increases the revision once events arrived from
[Provisioner](./the-internal-of-apisix-mesh-agent.md#Provisioner). the metadata will be filled according
to the point of time that the data set to cache.

Metadata might change after apisix-mesh-agent restarts, it's no matter since Apache APISIX will synchronize
from apisix-mesh-agent periodically.

## API Conformance

It's worth mentioning that not all features in `RangeRequest`, `WatchRequest` and others are supported,
only the used parts are implemented.

### RangeRequest

The implementation will check the `RangeRequest` from clients, if any unsupported features are touched,
error will be thrown.

* Only support exact key query or `readdir` styled range query are supported.

In terms of technology, the `range_end` field in `RangeRequest` should be `nil` (exact key query); Or
the query should be like `readdir` operation, since the key format of Apache APISIX is hierarchical,
the specific id component (`{id}` of `/apisix/routes/{id}`) should exist, for instance, the first following
`RangeRequest` is valid but the second one isn't.

```json
{
  "key": "/apisix/routes",
  "range_end": "apisix/routet"
}
```

```json
{
  "key": "/apisix/routes/1",
  "range_end": "apisix/routes/2"
}
```

* Limit the number of key-value pairs to return is not supported yet.

In terms of technology, the `limit` field in `RangeRequest` is ineffective, the implementation always
returns all key-value pairs.

* Data sorting is not supported yet.

In terms of technology, the `sort_order` and `sort_target` fields in `RangeRequest` are ineffective.

* No serializable and linearizable.

There is no concept of serializable and linearizable, because the solution is not a distributed system.
In terms of technology, the `serializable` field in `RangeRequest` is ignored.

* Specific revision is not supported yet.

In terms of technology, fields like `revision`, `min_mod_revision`, `max_mod_revision`, `min_create_revision`
and `max_create_revision` are ineffective.
