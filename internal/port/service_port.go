package port

import (
	"time"

	dbank "github.com/aditya3232/my-grpc-go-server/internal/application/domain/bank"
	"github.com/google/uuid"
)

type HelloServicePort interface {
	GenerateHello(name string) string
}

type BankServicePort interface {
	FindCurrentBalance(acct string) (float64, error)
	CreateExchangeRate(r dbank.ExchangeRate) (uuid.UUID, error)
	FindExchangeRate(fromCur string, toCur string, ts time.Time) (float64, error)
	CreateTransaction(acct string, t dbank.Transaction) (uuid.UUID, error)
	CalculateTransactionSummary(tcur *dbank.TransactionSummary, trans dbank.Transaction) error
	Transfer(tt dbank.TransferTransaction) (uuid.UUID, bool, error)
}
