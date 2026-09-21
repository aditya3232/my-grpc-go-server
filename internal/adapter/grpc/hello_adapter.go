package grpc

import (
	"context"
	"fmt"
	"time"

	"github.com/aditya3232/my-grpc-proto/protogen/go/hello"
	"google.golang.org/grpc"
)

// method grpc server
func (a *GrpcAdapter) SayHello(ctx context.Context, req *hello.HelloRequest) (*hello.HelloResponse, error) {
	greet := a.helloService.GenerateHello(req.Name)

	return &hello.HelloResponse{
		Greet: greet,
	}, nil
}

func (a *GrpcAdapter) SayManyHellos(req *hello.HelloRequest, stream grpc.ServerStreamingServer[hello.HelloResponse]) error {
	for i := range 10 {
		greet := a.helloService.GenerateHello(req.Name)
		res := fmt.Sprintf("[%d] %s", i, greet)

		stream.Send(
			&hello.HelloResponse{
				Greet: res,
			},
		)

		time.Sleep(500 * time.Millisecond)
	}

	return nil
}
