package database

import (
	"errors"
	"log"
	"time"

	dbank "github.com/aditya3232/my-grpc-go-server/internal/application/domain/bank"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (a *DatabaseAdapter) GetBankAccountByAccountNumber(acct string) (BankAccountOrm, error) {
	var bankAccountOrm BankAccountOrm

	if err := a.db.First(&bankAccountOrm, "account_number = ?", acct).Error; err != nil {
		log.Printf("Can't find bank account number %v : %v \n", acct, err)
		return bankAccountOrm, err
	}

	return bankAccountOrm, nil
}

func (a *DatabaseAdapter) CreateExchangeRate(r BankExchangeRateOrm) (uuid.UUID, error) {
	if err := a.db.Create(r).Error; err != nil {
		return uuid.Nil, err
	}

	return r.ExchangeRateUUID, nil
}

// GetExchangeRateAtTimestamp mengambil nilai tukar (exchange rate) dari fromCur ke toCur
// yang berlaku pada waktu ts, dengan mencocokkan ts terhadap rentang valid_from_timestamp
// dan valid_to_timestamp pada tabel bank_exchange_rate.
func (a *DatabaseAdapter) GetExchangeRateAtTimestamp(fromCur string, toCur string, ts time.Time) (BankExchangeRateOrm, error) {
	var exchangeRateOrm BankExchangeRateOrm

	err := a.db.
		Where("from_currency = ?", fromCur).
		Where("to_currency = ?", toCur).
		Where("? BETWEEN valid_from_timestamp AND valid_to_timestamp", ts).
		First(&exchangeRateOrm).Error

	return exchangeRateOrm, err
}

var ErrInsufficientBalance = errors.New("insufficient balance")

func (a *DatabaseAdapter) CreateTransaction(acct BankAccountOrm, t BankTransactionOrm) (uuid.UUID, error) {
	err := a.db.Transaction(func(tx *gorm.DB) error {
		// 1. Lock row akun dan ambil saldo terbaru dari DB
		var locked BankAccountOrm
		if err := tx.
			Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("account_uuid = ?", acct.AccountUUID). // sesuaikan dengan primary key kamu
			First(&locked).Error; err != nil {
			return err
		}

		// 2. Hitung saldo baru dari saldo yang sudah di-lock
		delta := t.Amount
		if t.TransactionType == dbank.TransactionTypeOut {
			delta = -t.Amount
		}

		newBalance := locked.CurrentBalance + delta
		if newBalance < 0 {
			return ErrInsufficientBalance // cek saldo negatif
		}

		// 3. Simpan transaksi (pakai pointer)
		if err := tx.Create(&t).Error; err != nil {
			return err
		}

		// 4. Update saldo
		return tx.Model(&locked).Updates(map[string]any{
			"current_balance": newBalance,
			"updated_at":      time.Now(),
		}).Error
	})
	if err != nil {
		return uuid.Nil, err
	}

	return t.TransactionUUID, nil
}
