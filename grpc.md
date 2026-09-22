# gRPC di Golang

gRPC (Google Remote Procedure Call) adalah framework RPC modern yang memungkinkan satu service memanggil fungsi di service lain seolah-olah fungsi tersebut lokal, meski sebenarnya berjalan di server/proses berbeda.

## Daftar Isi

- [1. Konsep Dasar](#1-konsep-dasar)
- [2. Alur Kerja Dasar](#2-alur-kerja-dasar)
- [3. Contoh File .proto](#3-contoh-file-proto)
- [4. Implementasi Server](#4-implementasi-server)
- [5. Implementasi Client](#5-implementasi-client)
- [6. Empat Tipe Komunikasi gRPC](#6-empat-tipe-komunikasi-grpc)
- [7. Kenapa Pakai gRPC di Golang?](#7-kenapa-pakai-grpc-di-golang)
- [Kapan Sebaiknya Pakai gRPC vs REST?](#kapan-sebaiknya-pakai-grpc-vs-rest)

---

## 1. Konsep Dasar

**RPC (Remote Procedure Call)**
Alih-alih membuat REST API dengan endpoint HTTP biasa, gRPC memungkinkan kamu mendefinisikan *service* dan *method* seperti memanggil fungsi biasa.

**Protocol Buffers (protobuf)**
gRPC menggunakan Protobuf sebagai *Interface Definition Language* (IDL) untuk mendefinisikan struktur data dan service. Ini menggantikan JSON dengan format binary yang lebih ringkas dan cepat.

**HTTP/2**
gRPC berjalan di atas HTTP/2, yang mendukung multiplexing (banyak request dalam satu koneksi), streaming dua arah, dan header compression — jauh lebih efisien dibanding HTTP/1.1 yang dipakai REST tradisional.

## 2. Alur Kerja Dasar

1. Definisikan service dan pesan di file `.proto`
2. Generate kode Go dari file `.proto` menggunakan `protoc`
3. Implementasikan server (logic bisnis)
4. Buat client yang memanggil service tersebut

## 3. Contoh File `.proto`

```protobuf
syntax = "proto3";

package greeter;
option go_package = "example.com/greeter";

service Greeter {
  rpc SayHello (HelloRequest) returns (HelloReply) {}
}

message HelloRequest {
  string name = 1;
}

message HelloReply {
  string message = 1;
}
```

Generate kode Go:

```bash
protoc --go_out=. --go-grpc_out=. greeter.proto
```

## 4. Implementasi Server

```go
package main

import (
    "context"
    "log"
    "net"

    "google.golang.org/grpc"
    pb "example.com/greeter"
)

type server struct {
    pb.UnimplementedGreeterServer
}

func (s *server) SayHello(ctx context.Context, req *pb.HelloRequest) (*pb.HelloReply, error) {
    return &pb.HelloReply{Message: "Hello, " + req.GetName()}, nil
}

func main() {
    lis, _ := net.Listen("tcp", ":50051")
    s := grpc.NewServer()
    pb.RegisterGreeterServer(s, &server{})
    log.Println("Server jalan di :50051")
    s.Serve(lis)
}
```

## 5. Implementasi Client

```go
package main

import (
    "context"
    "log"

    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"
    pb "example.com/greeter"
)

func main() {
    conn, _ := grpc.NewClient("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
    defer conn.Close()

    client := pb.NewGreeterClient(conn)
    resp, _ := client.SayHello(context.Background(), &pb.HelloRequest{Name: "Budi"})
    log.Println(resp.GetMessage())
}
```

## 6. Empat Tipe Komunikasi gRPC

| Tipe | Deskripsi |
|---|---|
| **Unary** | Satu request → satu response (seperti contoh di atas) |
| **Server streaming** | Satu request → banyak response (stream dari server) |
| **Client streaming** | Banyak request → satu response |
| **Bidirectional streaming** | Client dan server saling mengirim stream secara bersamaan |

## 7. Kenapa Pakai gRPC di Golang?

- **Performa tinggi** — binary protobuf + HTTP/2 lebih cepat dari JSON/REST
- **Strongly typed** — kontrak API jelas lewat `.proto`, mengurangi bug integrasi
- **Cocok untuk microservices** — komunikasi antar service internal jadi efisien
- **Streaming native** — cocok untuk use case real-time (chat, notifikasi, dsb)
- **Cross-language** — server di Go bisa dipanggil client Python, Java, dll

## Kapan Sebaiknya Pakai gRPC vs REST?

- **gRPC**: komunikasi antar microservice internal, butuh performa tinggi, streaming
- **REST**: API publik yang dikonsumsi browser/third-party, butuh human-readable format, atau butuh caching HTTP standar