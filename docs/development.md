Development
===========

This document explains how to get started to develop the apisix-mesh-agent.

Prerequisites
-------------

* You Go version should be at lease `1.14`.
* Clone the [apisix-mesh-agent](https://github.com/api7/apisix-mesh-agent) project.

Build
-----

```shell
cd /path/to/apisix-mesh-agent
make build
```

Test
----

### Run Unit Test Suites

```shell
cd /path/to/apisix-mesh-agent
make unit-test
```

### Mimic practical environment

If you want to mimic the practical environment, iptables rules should be set up in your development
environment, see [traffic-interception](./traffic-interception.md) for the details of creating
the iptables rules.
