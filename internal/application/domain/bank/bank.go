package bank

import (
	"errors"
	"time"
)

const (
	TransactionTypeUnknown string = "UNKNOWN"
	TransactionTypeIn      string = "IN"
	TransactionTypeOut     string = "OUT"
)

var (
	ErrTransferSourceAccountNotFound      = errors.New("source account not found")
	ErrTransferDestinationAccountNotFound = errors.New("destination account not found")
	ErrTransferSameAccount                = errors.New("source and destination account must be different")
	ErrTransferInvalidAmount              = errors.New("transfer amount must be greater than zero")
	ErrTransferInsufficientBalance        = errors.New("insufficient balance on source account")
	ErrTransferRecordFailed               = errors.New("can't create transfer record")
	ErrTransferTransactionPair            = errors.New("can't create transfer transaction pair")
)

type ExchangeRate struct {
	FromCurrency       string
	ToCurrency         string
	Rate               float64
	ValidFromTimestamp time.Time
	ValidToTimestamp   time.Time
}

type Transaction struct {
	Amount          float64
	Timestamp       time.Time
	TransactionType string
	Notes           string
}

type TransactionSummary struct {
	SummaryOnDate time.Time
	SumIn         float64
	SumOut        float64
	SumTotal      float64
}

type TransferTransaction struct {
	FromAccountNumber string
	ToAccountNumber   string
	Currency          string
	Amount            float64
}
