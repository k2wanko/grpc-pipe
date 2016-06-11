# gRPC Pipe

gprc-pipe is in memory net.Listner

# Install

```sh
go get -u github.com/golang/protobuf/protoc-gen-go
go get -u github.com/k2wanko/grpc-pipe
```

# Usage

## IDL

`echo_service.proto`

```protobuf
syntax = "proto3";
option go_package = "echo";

// Echo Service
//
// Echo Service API consists of a single service which returns
// a message.
package echo;

// SimpleMessage represents a simple message sent to the Echo service.
message Message {
    string value = 1;
}

// Echo service responds to incoming echo requests.
service EchoService {
    // Echo method receives a simple message and returns it.
    //
    // The message posted as the id parameter will also be
    // returned.
    rpc Echo(Message) returns (Message);
}
```

## generate

```sh
protoc -I/usr/local/include -I. \
-I$GOPATH/src \
--go_out=plugins=grpc:. \
echo_service.proto
```

## Implement

```go

import(
    "golang.org/x/net/context"
    pb "github.com/k2wanko/grpc-pipe/testdata/echo"
)

type EchoService struct{}

func (*EchoService) Echo(ctx context.Context, msg *pb.Message) (*pb.Message, error) {
  return msg, nil
}

```

## code

[link](pipe_test.go)

# Gateway

## Install

```sh
go get -u github.com/gengo/grpc-gateway/protoc-gen-grpc-gateway
go get -u github.com/gengo/grpc-gateway/protoc-gen-swagger
```

## IDL

```diff
package echo;

+import "google/api/annotations.proto";
+
// SimpleMessage represents a simple message sent to the Echo service.
message Message {
string value = 1;

//
// The message posted as the id parameter will also be
// returned.
-  rpc Echo(Message) returns (Message);
+  rpc Echo(Message) returns (Message) {
+    option (google.api.http) = {
+      post: "/echo"
+      body: "*"
+    };
+  }
 }
```

## generate

```sh
protoc -I/usr/local/include -I. \
-I$GOPATH/src \
-I$GOPATH/src/github.com/gengo/grpc-gateway/third_party/googleapis \
--swagger_out=logtostderr=true:. \
--grpc-gateway_out=logtostderr=true:. \
--go_out=Mgoogle/api/annotations.proto=github.com/gengo/grpc-gateway/third_party/googleapis/google/api,plugins=grpc:. \
echo_service.proto
```

## code

[link](gateway/server_test.go)
