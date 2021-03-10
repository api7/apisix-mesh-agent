module github.com/api7/apisix-mesh-agent

go 1.14

require (
	github.com/envoyproxy/go-control-plane v0.9.9-0.20210115003313-31f9241a16e6
	github.com/envoyproxy/protoc-gen-validate v0.4.1
	github.com/fsnotify/fsnotify v1.4.9
	github.com/golang/protobuf v1.4.3
	github.com/google/uuid v1.2.0
	github.com/grpc-ecosystem/grpc-gateway v1.14.6
	github.com/soheilhy/cmux v0.1.4
	github.com/spf13/cobra v1.1.3
	github.com/stretchr/testify v1.7.0
	github.com/tmc/grpc-websocket-proxy v0.0.0-20190109142713-0ad062ec5ee5
	go.etcd.io/etcd/api/v3 v3.5.0-alpha.0
	go.uber.org/zap v1.16.0
	golang.org/x/net v0.0.0-20210226172049-e18ecbb05110
	google.golang.org/genproto v0.0.0-20210222152913-aa3ee6e6a81c
	google.golang.org/grpc v1.36.0
	google.golang.org/grpc/examples v0.0.0-20210304020650-930c79186c99 // indirect
	google.golang.org/protobuf v1.25.0
	gotest.tools v2.2.0+incompatible
	istio.io/istio v0.0.0-20210308180034-f6502508b04c
	k8s.io/apimachinery v0.20.4
)
