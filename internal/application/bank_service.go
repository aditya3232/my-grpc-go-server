package application

import (
	"log"
	"time"

	"github.com/google/uuid"

	db "github.com/aditya3232/my-grpc-go-server/internal/adapter/database"
	dbank "github.com/aditya3232/my-grpc-go-server/internal/application/domain/bank"
	"github.com/aditya3232/my-grpc-go-server/internal/port"
)

type BankService struct {
	db port.BankDatabasePort
}

func NewBankService(dbPort port.BankDatabasePort) *BankService {
	return &BankService{
		db: dbPort,
	}
}

func (s *BankService) FindCurrentBalance(acct string) float64 {
	bankAccount, err := s.db.GetBankAccountByAccountNumber(acct)
	if err != nil {
		log.Println("Error on FindCurrentBalance : ", err)
	}

	return bankAccount.CurrentBalance
}

func (s *BankService) CreateExchangeRate(r dbank.ExchangeRate) (uuid.UUID, error) {
	newuuid := uuid.New()
	now := time.Now()

	exchangeRateOrm := db.BankExchangeRateOrm{
		ExchangeRateUUID:   newuuid,
		FromCurrency:       r.FromCurrency,
		ToCurrency:         r.ToCurrency,
		Rate:               r.Rate,
		ValidFromTimestamp: r.ValidFromTimestamp,
		ValidToTimestamp:   r.ValidToTimestamp,
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	return s.db.CreateExchangeRate(exchangeRateOrm)
}

func (s *BankService) FindExchangeRate(fromCur string, toCur string, ts time.Time) float64 {
	exchangeRate, err := s.db.GetExchangeRateAtTimestamp(fromCur, toCur, ts)
	if err != nil {
		return 0
	}

	return float64(exchangeRate.Rate)
}
