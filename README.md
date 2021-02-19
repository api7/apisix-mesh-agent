<!--
#
# Licensed to the Apache Software Foundation (ASF) under one or more
# contributor license agreements.  See the NOTICE file distributed with
# this work for additional information regarding copyright ownership.
# The ASF licenses this file to You under the Apache License, Version 2.0
# (the "License"); you may not use this file except in compliance with
# the License.  You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
-->

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
