package grpc

import (
	"context"

	"github.com/aditya3232/my-grpc-proto/protogen/go/hello"
)

// method grpc server
func (a *GrpcAdapter) SayHello(ctx context.Context, req *hello.HelloRequest) (*hello.HelloResponse, error) {
	greet := a.helloService.GenerateHello(req.Name)

	return &hello.HelloResponse{
		Greet: greet,
	}, nil
}
