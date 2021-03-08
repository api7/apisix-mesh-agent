module github.com/api7/apisix-mesh-agent

go 1.14

require (
	github.com/envoyproxy/go-control-plane v0.9.8
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
	go.uber.org/multierr v1.6.0 // indirect
	go.uber.org/zap v1.16.0
	golang.org/x/net v0.0.0-20201110031124-69a78807bb2b
	google.golang.org/genproto v0.0.0-20200806141610-86f49bd18e98
	google.golang.org/grpc v1.32.0
	google.golang.org/grpc/examples v0.0.0-20210304020650-930c79186c99 // indirect
	google.golang.org/protobuf v1.25.0
	gotest.tools v2.2.0+incompatible
	k8s.io/apimachinery v0.20.4
)
