apisix-mesh-agent
=================

Agent of [Apache APISIX](http://apisix.apache.org/) to extend it as a [Service
Mesh](https://www.redhat.com/en/topics/microservices/what-is-a-service-mesh) Sidecar.

Status
------

This project is currently considered as experimental.

Why apisix-mesh-agent
---------------------

APISIX provides rich traffic management features such as load balancing, dynamic upstream, canary release, circuit breaking, authentication, observability, and more.

It's an excellent API Gateway but is not sufficient for Service Mesh, with the help of apisix-mesh-agent, it handles the East-West traffic well.

The Design of APISIX Mesh
-------------------------

See the [Design](./docs/design.md) for the details.

License
-------

[Apache 2.0 LICENSE](./LICENSE)
