package grpc

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
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

func (a *GrpcAdapter) SayHelloToEveryone(stream grpc.ClientStreamingServer[hello.HelloRequest, hello.HelloResponse]) error {
	res := ""

	for {
		req, err := stream.Recv()
		switch {
		// setelah client selesai mengirim data, server lalu mengirim response tunggal berisi res dengan SendAndClose
		case errors.Is(err, io.EOF):
			return stream.SendAndClose(
				&hello.HelloResponse{
					Greet: res,
				},
			)
		case err != nil:
			log.Fatalln("Error while reading from client : ", err)
		}

		greet := a.helloService.GenerateHello(req.Name)

		res += greet + " "
	}

}

func (a *GrpcAdapter) SayHelloContinous(stream grpc.BidiStreamingServer[hello.HelloRequest, hello.HelloResponse]) error {
	for {
		req, err := stream.Recv()
		switch {
		case errors.Is(err, io.EOF):
			return nil
		case err != nil:
			log.Fatalln("Error while reading from client : ", err)
		}

		// untuk setiap request kita membuat string response dan mengirimkan response ke client
		greet := a.helloService.GenerateHello(req.Name)

		err = stream.Send(
			&hello.HelloResponse{
				Greet: greet,
			},
		)
		if err != nil {
			log.Fatalln("Error while sending response to client : ", err)
		}
	}
}
