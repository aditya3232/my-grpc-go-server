package database

import (
	"log"
	"time"

	"github.com/google/uuid"
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
