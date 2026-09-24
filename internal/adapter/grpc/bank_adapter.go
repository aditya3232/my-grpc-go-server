package grpc

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/google/uuid"

	"github.com/aditya3232/my-grpc-proto/protogen/go/bank"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/genproto/googleapis/type/date"
	"google.golang.org/genproto/googleapis/type/datetime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	dbank "github.com/aditya3232/my-grpc-go-server/internal/application/domain/bank"
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

func toTime(dt *datetime.DateTime) (time.Time, error) {
	if dt == nil {
		now := time.Now()

		dt = &datetime.DateTime{
			Year:    int32(now.Year()),
			Month:   int32(now.Month()),
			Day:     int32(now.Day()),
			Hours:   int32(now.Hour()),
			Minutes: int32(now.Minute()),
			Seconds: int32(now.Second()),
			Nanos:   int32(now.Nanosecond()),
		}
	}

	res := time.Date(int(dt.Year), time.Month(dt.Month), int(dt.Day),
		int(dt.Hours), int(dt.Minutes), int(dt.Seconds), int(dt.Nanos), time.UTC)

	return res, nil
}

// SummarizeTransactions adalah handler gRPC client-streaming: client mengirim
// banyak Transaction, lalu server membalas SATU TransactionSummary setelah
// client selesai mengirim (ditandai io.EOF).
func (a *GrpcAdapter) SummarizeTransactions(stream grpc.ClientStreamingServer[bank.Transaction, bank.TransactionSummary]) error {
	// Akumulator ringkasan. Diisi bertahap setiap kali satu transaksi diterima.
	tsum := dbank.TransactionSummary{
		SummaryOnDate: time.Now(),
		SumIn:         0,
		SumOut:        0,
		SumTotal:      0,
	}
	// Nomor rekening disimpan di luar loop supaya bisa dipakai saat
	// membentuk response akhir. Nilainya ditimpa tiap pesan masuk.
	acct := ""

	// Loop membaca pesan dari client sampai stream ditutup.
	for {
		// Recv() memblokir sampai ada pesan berikutnya dari client.
		req, err := stream.Recv()

		// io.EOF berarti client sudah selesai mengirim semua transaksi.
		// Saatnya membangun response ringkasan dan menutup stream.
		if err == io.EOF {
			res := bank.TransactionSummary{
				AccountNumber: acct,
				SumAmountIn:   tsum.SumIn,
				SumAmountOut:  tsum.SumOut,
				SumTotal:      tsum.SumTotal,
				// Tanggal ringkasan diubah ke format google.type.Date
				// (hanya tahun/bulan/hari, tanpa jam).
				TransactionDate: &date.Date{
					Year:  int32(tsum.SummaryOnDate.Year()),
					Month: int32(tsum.SummaryOnDate.Month()),
					Day:   int32(tsum.SummaryOnDate.Day()),
				},
			}

			// Kirim response tunggal lalu tutup stream dari sisi server.
			return stream.SendAndClose(&res)
		}

		// Error selain EOF (koneksi putus, context dibatalkan, dsb).
		if err != nil {
			log.Fatalln("Error while reading from client :", err)
		}

		// Catat nomor rekening dari transaksi terbaru.
		acct = req.AccountNumber

		// Konversi timestamp dari request ke time.Time.
		ts, err := toTime(req.Timestamp)

		if err != nil {
			log.Fatalf("Error while parsing timestamp %v : %v", req.Timestamp, err)
		}

		// Petakan enum protobuf ke tipe transaksi domain.
		// Defaultnya Unknown, jadi tipe di luar IN/OUT tidak akan salah dikenali.
		ttype := dbank.TransactionTypeUnknown

		switch req.Type {
		case bank.TransactionType_TRANSACTION_TYPE_IN:
			ttype = dbank.TransactionTypeIn
		case bank.TransactionType_TRANSACTION_TYPE_OUT:
			ttype = dbank.TransactionTypeOut
		}

		// Bentuk objek transaksi domain dari data request.
		tcur := dbank.Transaction{
			Amount:          req.Amount,
			Timestamp:       ts,
			TransactionType: ttype,
		}

		// Simpan transaksi ke database lewat service layer
		// (di dalamnya mengubah saldo rekening juga).
		accountUuid, err := a.bankService.CreateTransaction(req.AccountNumber, tcur)

		// Kasus 1: gagal DAN UUID kosong. Diasumsikan rekening tidak ditemukan,
		// jadi dibalas InvalidArgument dengan detail field "account_number".
		if err != nil && accountUuid == uuid.Nil {
			s := status.New(codes.InvalidArgument, err.Error())
			s, _ = s.WithDetails(&errdetails.BadRequest{
				FieldViolations: []*errdetails.BadRequest_FieldViolation{
					{
						Field:       "account_number",
						Description: "Invalid account number",
					},
				},
			})

			return s.Err()
			// Kasus 2: gagal TAPI UUID terisi. Rekening ada, jadi diasumsikan
			// penyebabnya nominal melebihi saldo, dengan detail field "amount".
		} else if err != nil && accountUuid != uuid.Nil {
			s := status.New(codes.InvalidArgument, err.Error())
			s, _ = s.WithDetails(&errdetails.BadRequest{
				FieldViolations: []*errdetails.BadRequest_FieldViolation{
					{
						Field:       "amount",
						Description: fmt.Sprintf("Requested amount %v exceed available balance", req.Amount),
					},
				},
			})

			return s.Err()
		}

		// Sisa error yang belum tertangani dicatat ke log saja.
		if err != nil {
			log.Println("Error while creating transaction :", err)
		}

		// Tambahkan transaksi ini ke akumulator ringkasan
		// (menambah SumIn/SumOut dan menghitung SumTotal).
		err = a.bankService.CalculateTransactionSummary(&tsum, tcur)

		if err != nil {
			return err
		}
	}
}

func currentDatetime() *datetime.DateTime {
	now := time.Now().UTC()

	return &datetime.DateTime{
		Year:       int32(now.Year()),
		Month:      int32(now.Month()),
		Day:        int32(now.Day()),
		Hours:      int32(now.Hour()),
		Minutes:    int32(now.Minute()),
		Seconds:    int32(now.Second()),
		Nanos:      int32(now.Nanosecond()),
		TimeOffset: &datetime.DateTime_UtcOffset{},
	}
}

func buildTransferErrorStatusGrpc(err error, req *bank.TransferRequest) error {
	switch {
	case errors.Is(err, dbank.ErrTransferSourceAccountNotFound):
		s := status.New(codes.FailedPrecondition, err.Error())
		s, _ = s.WithDetails(&errdetails.PreconditionFailure{
			Violations: []*errdetails.PreconditionFailure_Violation{
				{
					Type:        "INVALID_ACCOUNT",
					Subject:     "Source account not found",
					Description: fmt.Sprintf("source account (from %v) not found", req.FromAccountNumber),
				},
			},
		})
		return s.Err()

	case errors.Is(err, dbank.ErrTransferDestinationAccountNotFound):
		s := status.New(codes.FailedPrecondition, err.Error())
		s, _ = s.WithDetails(&errdetails.PreconditionFailure{
			Violations: []*errdetails.PreconditionFailure_Violation{
				{
					Type:        "INVALID_ACCOUNT",
					Subject:     "Destination account not found",
					Description: fmt.Sprintf("destination account (to %v) not found", req.ToAccountNumber),
				},
			},
		})
		return s.Err()

	case errors.Is(err, dbank.ErrTransferInsufficientBalance):
		s := status.New(codes.FailedPrecondition, err.Error())
		s, _ = s.WithDetails(&errdetails.PreconditionFailure{
			Violations: []*errdetails.PreconditionFailure_Violation{
				{
					Type:        "INSUFFICIENT_BALANCE",
					Subject:     "Insufficient balance",
					Description: fmt.Sprintf("balance on account %v is not enough to transfer %v", req.FromAccountNumber, req.Amount),
				},
			},
		})
		return s.Err()

	case errors.Is(err, dbank.ErrTransferInvalidAmount):
		s := status.New(codes.InvalidArgument, err.Error())
		s, _ = s.WithDetails(&errdetails.BadRequest{
			FieldViolations: []*errdetails.BadRequest_FieldViolation{
				{Field: "amount", Description: "amount must be greater than zero"},
			},
		})
		return s.Err()

	case errors.Is(err, dbank.ErrTransferSameAccount):
		s := status.New(codes.InvalidArgument, err.Error())
		s, _ = s.WithDetails(&errdetails.BadRequest{
			FieldViolations: []*errdetails.BadRequest_FieldViolation{
				{Field: "to_account_number", Description: "destination must differ from source account"},
			},
		})
		return s.Err()

	case errors.Is(err, dbank.ErrTransferRecordFailed):
		s := status.New(codes.Internal, err.Error())
		s, _ = s.WithDetails(&errdetails.Help{
			Links: []*errdetails.Help_Link{
				{Url: "my-bank-website.com/faq", Description: "Bank FAQ"},
			},
		})
		return s.Err()

	case errors.Is(err, dbank.ErrTransferTransactionPair):
		s := status.New(codes.Internal, err.Error())
		s, _ = s.WithDetails(&errdetails.ErrorInfo{
			Domain: "my-bank-website.com",
			Reason: "TRANSACTION_PAIR_FAILED",
			Metadata: map[string]string{
				"from_account": req.FromAccountNumber,
				"to_account":   req.ToAccountNumber,
				"currency":     req.Currency,
				"amount":       fmt.Sprintf("%v", req.Amount),
			},
		})
		return s.Err()

	default:
		// Jangan bocorkan pesan error internal ke client
		log.Println("Unhandled transfer error :", err)
		return status.Error(codes.Internal, "internal error")
	}
}

func (a *GrpcAdapter) TransferMultiple(stream bank.BankService_TransferMultipleServer) error {
	for {
		// Recv() otomatis mengembalikan error kalau context dibatalkan,
		// jadi tidak perlu select pada ctx.Done() secara terpisah.
		req, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			log.Println("Error while reading from client :", err)
			return err
		}

		tt := dbank.TransferTransaction{
			FromAccountNumber: req.FromAccountNumber,
			ToAccountNumber:   req.ToAccountNumber,
			Currency:          req.Currency,
			Amount:            req.Amount,
		}

		_, transferSuccess, err := a.bankService.Transfer(tt)
		if err != nil {
			return buildTransferErrorStatusGrpc(err, req)
		}

		res := &bank.TransferResponse{
			FromAccountNumber: req.FromAccountNumber,
			ToAccountNumber:   req.ToAccountNumber,
			Currency:          req.Currency,
			Amount:            req.Amount,
			Timestamp:         currentDatetime(),
			Status:            bank.TransferStatus_TRANSFER_STATUS_FAILED,
		}

		if transferSuccess {
			res.Status = bank.TransferStatus_TRANSFER_STATUS_SUCCESS
		}

		if err := stream.Send(res); err != nil {
			log.Println("Error while sending response to client :", err)
			return err
		}
	}
}
