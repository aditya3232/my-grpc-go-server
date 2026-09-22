package port

import (
	"time"

	db "github.com/aditya3232/my-grpc-go-server/internal/adapter/database"
	"github.com/google/uuid"
)

type DummyDatabasePort interface {
	Save(data *db.DummyOrm) (uuid.UUID, error)
	GetByUUID(uuid *uuid.UUID) (db.DummyOrm, error)
}

type BankDatabasePort interface {
	GetBankAccountByAccountNumber(acct string) (db.BankAccountOrm, error)
	CreateExchangeRate(r db.BankExchangeRateOrm) (uuid.UUID, error)                                        // buat data dummy exchange rate
	GetExchangeRateAtTimestamp(fromCur string, toCur string, ts time.Time) (db.BankExchangeRateOrm, error) // ambil kurs berdasarkan mata uang di waktu tertentu
}
