package application

import (
	"errors"
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

func (s *BankService) FindCurrentBalance(acct string) (float64, error) {
	bankAccount, err := s.db.GetBankAccountByAccountNumber(acct)
	if err != nil {
		log.Println("Error on FindCurrentBalance : ", err)
		return 0, err
	}

	return bankAccount.CurrentBalance, nil
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

func (s *BankService) Transfer(tt dbank.TransferTransaction) (uuid.UUID, bool, error) {
	// 1. Validasi input sebelum menyentuh database
	if tt.Amount <= 0 {
		return uuid.Nil, false, dbank.ErrTransferInvalidAmount
	}

	if tt.FromAccountNumber == tt.ToAccountNumber {
		return uuid.Nil, false, dbank.ErrTransferSameAccount
	}

	// 2. Ambil kedua rekening
	fromAccountOrm, err := s.db.GetBankAccountByAccountNumber(tt.FromAccountNumber)
	if err != nil {
		log.Printf("Can't find transfer from account %v : %v\n", tt.FromAccountNumber, err)
		return uuid.Nil, false, dbank.ErrTransferSourceAccountNotFound
	}

	toAccountOrm, err := s.db.GetBankAccountByAccountNumber(tt.ToAccountNumber)
	if err != nil {
		log.Printf("Can't find transfer to account %v : %v\n", tt.ToAccountNumber, err)
		return uuid.Nil, false, dbank.ErrTransferDestinationAccountNotFound
	}

	// 3. Catat permintaan transfer (status awal: belum sukses)
	now := time.Now()

	transferOrm := db.BankTransferOrm{
		TransferUUID:      uuid.New(),
		FromAccountUUID:   fromAccountOrm.AccountUUID,
		ToAccountUUID:     toAccountOrm.AccountUUID,
		Currency:          tt.Currency,
		Amount:            tt.Amount,
		TransferTimestamp: now,
		TransferSuccess:   false,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	if _, err := s.db.CreateTransfer(transferOrm); err != nil {
		log.Printf("Can't create transfer from %v to %v : %v\n", tt.FromAccountNumber, tt.ToAccountNumber, err)
		return uuid.Nil, false, dbank.ErrTransferRecordFailed
	}

	// 4. Siapkan pasangan transaksi: uang keluar (pengirim) dan uang masuk (penerima)
	fromTransactionOrm := db.BankTransactionOrm{
		TransactionUUID:      uuid.New(),
		TransactionTimestamp: now,
		TransactionType:      dbank.TransactionTypeOut,
		AccountUUID:          fromAccountOrm.AccountUUID,
		Amount:               tt.Amount,
		Notes:                "Transfer out to " + tt.ToAccountNumber,
		CreatedAt:            now,
		UpdatedAt:            now,
	}

	toTransactionOrm := db.BankTransactionOrm{
		TransactionUUID:      uuid.New(),
		TransactionTimestamp: now,
		TransactionType:      dbank.TransactionTypeIn,
		AccountUUID:          toAccountOrm.AccountUUID,
		Amount:               tt.Amount,
		Notes:                "Transfer in from " + tt.FromAccountNumber,
		CreatedAt:            now,
		UpdatedAt:            now,
	}

	// 5. Eksekusi transfer secara atomik (cek saldo dilakukan di dalam DB transaction)
	if _, err := s.db.CreateTransferTransactionPair(fromAccountOrm, toAccountOrm, fromTransactionOrm, toTransactionOrm); err != nil {
		log.Printf("Can't create transfer transaction pair from %v to %v : %v\n",
			tt.FromAccountNumber, tt.ToAccountNumber, err)

		if errors.Is(err, db.ErrInsufficientBalance) {
			return transferOrm.TransferUUID, false, dbank.ErrTransferInsufficientBalance
		}

		return transferOrm.TransferUUID, false, dbank.ErrTransferTransactionPair
	}

	// 6. Tandai transfer sukses. Uang sudah berpindah, jadi kegagalan di sini
	// hanya dicatat di log, bukan dikembalikan sebagai kegagalan transfer.
	if err := s.db.UpdateTransferStatus(transferOrm, true); err != nil {
		log.Printf("Transfer %v succeeded but status update failed : %v\n", transferOrm.TransferUUID, err)
	}

	return transferOrm.TransferUUID, true, nil
}
