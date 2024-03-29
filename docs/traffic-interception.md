# Traffic Interception

This document explains how apisix-mesh-agent intercepts TCP traffic to [Apache APISIX](https://apisix.apache.org).

A Sidecar program must have the ability to intercept TCP traffic which would be sent from/to the application, and the
process should be opaque, without any changes made by user's application.

## Iptables

apisix-mesh-agent sets up [Iptables](https://en.wikipedia.org/wiki/Iptables) rules to forward both inbound
and outbound TCP traffic to APISIX ports (e.g. `9080` for outbound and `9081` for inbound).

Iptables rules should be set up when the Pod/VM initialized. What's more, super user permission should be
assigned when setting up these rules.

## Command

The `apisix-mesh-agent` utility provides a pair of commands to set and clean iptables rules.

The `apisix-mesh-agent iptables` command creates several iptables rules,
see `apisix-mesh-agent iptables --help` if you want to know each option's mean.

The `apisix-mesh-agent cleanup-iptables` command clears rules created previously, see `apisix-mesh-agent cleanup-iptables --help`
if you want to know each option's mean.

### Dry run mode

It's recommended to use dry run mode firstly to see which commands will be executed. By specifying
`--dry-run` option, both the `iptables` and `cleanup-iptables` subcommand won't make effect but only output
rules.

### Examples

1. Forward all inbound TCP traffic to port `9081`.

```shell
./apisix-mesh-agent iptables --apisix-inbound-capture-port 9081 --apisix-user root --dry-run
iptables -t nat -N APISIX_REDIRECT
iptables -t nat -N APISIX_INBOUND_REDIRECT
iptables -t nat -A APISIX_REDIRECT -p tcp -j REDIRECT --to-ports 9080
iptables -t nat -A APISIX_INBOUND_REDIRECT -p tcp -j REDIRECT --to-ports 9081
iptables -t nat -A OUTPUT -o lo ! -d 127.0.0.1/32 -m owner --uid-owner 0 -j RETURN
iptables -t nat -A OUTPUT -m owner --gid-owner 0 -j RETURN
```

Note the `--uid-owner` and `--gid-owner` values might be different, it depends on which user you specified to run the proxy component.

2. Forward inbound TCP traffic to port `9080` if the original destination port is `80` or `443`

```shell
./apisix-mesh-agent iptables --apisix-port 9080 --inbound-ports 80,443 --apisix-user root --dry-run
iptables -t nat -N APISIX_REDIRECT
iptables -t nat -N APISIX_INBOUND_REDIRECT
iptables -t nat -N APISIX_INBOUND
iptables -t nat -A APISIX_REDIRECT -p tcp -j REDIRECT --to-ports 9080
iptables -t nat -A APISIX_INBOUND_REDIRECT -p tcp -j REDIRECT --to-ports 9081
iptables -t nat -A OUTPUT -o lo ! -d 127.0.0.1/32 -m owner --uid-owner 0 -j RETURN
iptables -t nat -A OUTPUT -m owner --gid-owner 0 -j RETURN
iptables -t nat -A PREROUTING -p tcp -j APISIX_INBOUND
iptables -t nat -A APISIX_INBOUND -p tcp --dport 80 -j APISIX_INBOUND_REDIRECT
iptables -t nat -A APISIX_INBOUND -p tcp --dport 443 -j APISIX_INBOUND_REDIRECT
```

Note the `--uid-owner` and `--gid-owner` values might be different, it depends on which user you specified to run the proxy component.

2. Forward inbound TCP traffic to port `9081` if the original destination port is `80` or `443`

```shell
./apisix-mesh-agent iptables --apisix-inbound-capture-port 9081 --inbound-ports 80,443 --dry-run
iptables -t nat -N APISIX_REDIRECT
iptables -t nat -N APISIX_INBOUND_REDIRECT
iptables -t nat -N APISIX_INBOUND
iptables -t nat -A APISIX_REDIRECT -p tcp -j REDIRECT --to-ports 9080
iptables -t nat -A APISIX_INBOUND_REDIRECT -p tcp -j REDIRECT --to-ports 9081
iptables -t nat -A OUTPUT -o lo ! -d 127.0.0.1/32 -m owner --uid-owner 4294967294 -j RETURN
iptables -t nat -A OUTPUT -m owner --gid-owner 4294967294 -j RETURN
iptables -t nat -A PREROUTING -p tcp -j APISIX_INBOUND
iptables -t nat -A APISIX_INBOUND -p tcp --dport 80 -j APISIX_INBOUND_REDIRECT
iptables -t nat -A APISIX_INBOUND -p tcp --dport 443 -j APISIX_INBOUND_REDIRECT
```

Note the `--uid-owner` and `--gid-owner` values might be different, it depends on which user you specified to run the proxy component.

3. Forward outbound TCP to port `9080` if the original destination port is `80`

```shell
./apisix-mesh-agent iptables --apisix-port 9080 --dry-run --outbound-ports 80 --apisix-user root
iptables -t nat -N APISIX_REDIRECT
iptables -t nat -N APISIX_INBOUND_REDIRECT
iptables -t nat -A APISIX_REDIRECT -p tcp -j REDIRECT --to-ports 9080
iptables -t nat -A APISIX_INBOUND_REDIRECT -p tcp -j REDIRECT --to-ports 9081
iptables -t nat -A OUTPUT -o lo ! -d 127.0.0.1/32 -m owner --uid-owner 0 -j RETURN
iptables -t nat -A OUTPUT -m owner --gid-owner 0 -j RETURN
iptables -t nat -A OUTPUT -p tcp --dport 80 -j APISIX_REDIRECT
```

Note the `--uid-owner` and `--gid-owner` values might be different, it depends on which user you specified to run the proxy component.

4. Combination of 2 and 3

```shell
./apisix-mesh-agent iptables --apisix-inbound-capture-port 9081 --apisix-port 9080 --inbound-ports 80,443 --outbound-ports 80 --dry-run
iptables -t nat -N APISIX_REDIRECT
iptables -t nat -N APISIX_INBOUND_REDIRECT
iptables -t nat -N APISIX_INBOUND
iptables -t nat -A APISIX_REDIRECT -p tcp -j REDIRECT --to-ports 9080
iptables -t nat -A APISIX_INBOUND_REDIRECT -p tcp -j REDIRECT --to-ports 9081
iptables -t nat -A OUTPUT -o lo ! -d 127.0.0.1/32 -m owner --uid-owner 4294967294 -j RETURN
iptables -t nat -A OUTPUT -m owner --gid-owner 4294967294 -j RETURN
iptables -t nat -A PREROUTING -p tcp -j APISIX_INBOUND
iptables -t nat -A APISIX_INBOUND -p tcp --dport 80 -j APISIX_INBOUND_REDIRECT
iptables -t nat -A APISIX_INBOUND -p tcp --dport 443 -j APISIX_INBOUND_REDIRECT
iptables -t nat -A OUTPUT -p tcp --dport 80 -j APISIX_REDIRECT
```

Note the `--uid-owner` and `--gid-owner` values might be different, it depends on which user you specified to run the proxy component.

5. Forward all inbound traffic except port `2379`

```shell
./apisix-mesh-agent iptables --dry-run --inbound-ports * --inbound-exclude-ports 2379
iptables -t nat -N APISIX_REDIRECT
iptables -t nat -N APISIX_INBOUND_REDIRECT
iptables -t nat -N APISIX_INBOUND
iptables -t nat -A APISIX_REDIRECT -p tcp -j REDIRECT --to-ports 9080
iptables -t nat -A APISIX_INBOUND_REDIRECT -p tcp -j REDIRECT --to-ports 9081
iptables -t nat -A OUTPUT -o lo ! -d 127.0.0.1/32 -m owner --uid-owner 4294967294 -j RETURN
iptables -t nat -A OUTPUT -m owner --gid-owner 4294967294 -j RETURN
iptables -t nat -A PREROUTING -p tcp -j APISIX_INBOUND
iptables -t nat -A APISIX_INBOUND -p tcp --dport 22 -j RETURN
iptables -t nat -A APISIX_INBOUND -p tcp --dport 2379 -j RETURN
iptables -t nat -A APISIX_INBOUND -p tcp -j APISIX_INBOUND_REDIRECT
```

Note the `--uid-owner` and `--gid-owner` values might be different, it depends on which user you specified to run the proxy component.

6. Forward all outbound traffic except port `15010`

```shell
./apisix-mesh-agent iptables --dry-run --outbound-ports * --outbound-exclude-ports 15010
iptables -t nat -N APISIX_REDIRECT
iptables -t nat -N APISIX_INBOUND_REDIRECT
iptables -t nat -A APISIX_REDIRECT -p tcp -j REDIRECT --to-ports 9080
iptables -t nat -A APISIX_INBOUND_REDIRECT -p tcp -j REDIRECT --to-ports 9081
iptables -t nat -A OUTPUT -o lo ! -d 127.0.0.1/32 -m owner --uid-owner 4294967294 -j RETURN
iptables -t nat -A OUTPUT -m owner --gid-owner 4294967294 -j RETURN
iptables -t nat -A OUTPUT -p tcp --dport 15010 -j RETURN
iptables -t nat -A OUTPUT -p tcp -j APISIX_REDIRECT
```

7. Cleanup rules

```shell
./apisix-mesh-agent cleanup-iptables --dry-run
iptables -t nat -D PREROUTING -p tcp -j APISIX_INBOUND
iptables -t nat -D OUTPUT -p tcp -j OUTPUT
iptables -t nat -F APISIX_INBOUND
iptables -t nat -X APISIX_INBOUND
iptables -t nat -F OUTPUT
iptables -t nat -X OUTPUT
iptables -t nat -F APISIX_REDIRECT
iptables -t nat -X APISIX_REDIRECT
iptables -t nat -F APISIX_INBOUND_REDIRECT
iptables -t nat -X APISIX_INBOUND_REDIRECT
```
