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
	CreateTransaction(acct db.BankAccountOrm, t db.BankTransactionOrm) (uuid.UUID, error)

	// berhubungan dengan bidirectional stream
	CreateTransfer(transfer db.BankTransferOrm) (uuid.UUID, error) // memasukkan request transfer uang ke db
	CreateTransferTransactionPair(fromAccountOrm db.BankAccountOrm, toAccountOrm db.BankAccountOrm,
		fromTransactionOrm db.BankTransactionOrm, toTransactionOrm db.BankTransactionOrm) (bool, error) // buat pasangan transaksi, yaitu membuat transaksi uang keluar di rekening pengirim dan buat transaksi uang masuk di rekening penerima, kemudian mengupdate saldo kedua rekening
	UpdateTransferStatus(transfer db.BankTransferOrm, status bool) error // update status bank transfer, apakah transfer sukses atau gagal
}
