package grpc

import (
	"context"
	"log"
	"time"

	"github.com/aditya3232/my-grpc-proto/protogen/go/bank"
	"google.golang.org/genproto/googleapis/type/date"
	"google.golang.org/grpc"
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

func (a *GrpcAdapter) FetchExchangeRates(req *bank.ExchangeRateRequest, stream grpc.ServerStreamingServer[bank.ExchangeRateResponse]) error {
	context := stream.Context()

	for {
		select {
		case <-context.Done():
			log.Println("Client cancelled stream")
			return nil
		default:
			now := time.Now()
			rate := a.bankService.FindExchangeRate(req.FromCurrency, req.ToCurrency, now)

			stream.Send(
				&bank.ExchangeRateResponse{
					FromCurrency: req.FromCurrency,
					ToCurrency:   req.ToCurrency,
					Rate:         rate,
					Timestamp:    now.Format(time.RFC3339),
				},
			)

			log.Printf("Exchange rate sent to client, %v to %v : %v \n", req.FromCurrency, req.ToCurrency, rate)

			time.Sleep(3 * time.Second)
		}
	}
}
