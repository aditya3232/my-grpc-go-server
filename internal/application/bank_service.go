package application

import (
	"fmt"
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

func (s *BankService) CreateTransaction(acct string, t dbank.Transaction) (uuid.UUID, error) {
	newUUID := uuid.New()
	now := time.Now()

	bankAccountOrm, err := s.db.GetBankAccountByAccountNumber(acct)
	if err != nil {
		log.Printf("Can't create transaction for %v : %v \n", acct, err)
		return uuid.Nil, err
	}

	transactionOrm := db.BankTransactionOrm{
		TransactionUUID:      newUUID,
		AccountUUID:          bankAccountOrm.AccountUUID,
		TransactionTimestamp: now,
		Amount:               t.Amount,
		TransactionType:      t.TransactionType,
		Notes:                t.Notes,
		CreatedAt:            now,
		UpdatedAt:            now,
	}

	savedUUID, err := s.db.CreateTransaction(bankAccountOrm, transactionOrm)
	if err != nil {
		return uuid.Nil, err
	}

	return savedUUID, nil
}

func (s *BankService) CalculateTransactionSummary(tcur *dbank.TransactionSummary, trans dbank.Transaction) error {
	switch trans.TransactionType {
	case dbank.TransactionTypeIn:
		tcur.SumIn += trans.Amount
	case dbank.TransactionTypeOut:
		tcur.SumOut += trans.Amount
	default:
		return fmt.Errorf("unknown transaction type %v", trans.TransactionType)
	}

	tcur.SumTotal = tcur.SumIn - tcur.SumOut

	return nil
}
