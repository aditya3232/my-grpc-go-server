package grpc

import (
	"context"
	"time"

	"github.com/aditya3232/my-grpc-proto/protogen/go/bank"
	"google.golang.org/genproto/googleapis/type/date"
)

/*
Untuk tes dipostman:
- pilih service.proto dari my-grpc-proto
- import paths nya juga, karena ada "google.golang.org/genproto/googleapis/type/date" -> import path lokasi folder my-grpc-proto
- setelah itu baru bisa pilih method nya
*/
func (a *GrpcAdapter) GetCurrentBalance(ctx context.Context, req *bank.CurrentBalanceRequest) (*bank.CurrentBalanceResponse, error) {
	now := time.Now()
	bal := a.bankService.FindCurrentBalance(req.AccoutNumber)

	return &bank.CurrentBalanceResponse{
		Amount: bal,
		CurrentDate: &date.Date{
			Year:  int32(now.Year()),
			Month: int32(now.Month()),
			Day:   int32(now.Day()),
		},
	}, nil
}
