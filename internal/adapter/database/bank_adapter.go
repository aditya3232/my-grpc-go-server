package database

import (
	"errors"
	"log"
	"sort"
	"time"

	dbank "github.com/aditya3232/my-grpc-go-server/internal/application/domain/bank"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrInsufficientBalance = errors.New("insufficient balance")
	ErrSameAccountTransfer = errors.New("cannot transfer to the same account")
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

func (a *DatabaseAdapter) CreateTransfer(transfer BankTransferOrm) (uuid.UUID, error) {
	if err := a.db.Create(&transfer).Error; err != nil {
		return uuid.Nil, err
	}

	return transfer.TransferUUID, nil
}

// CreateTransferTransactionPair memindahkan dana dari rekening pengirim ke
// rekening penerima secara atomik, dalam satu database transaction.
//
// Langkah yang dijalankan:
//  1. Mengunci (SELECT ... FOR UPDATE) kedua rekening dengan urutan yang
//     konsisten untuk mencegah deadlock dan race condition.
//  2. Membaca saldo terbaru dari database. Field CurrentBalance pada
//     fromAccountOrm dan toAccountOrm diabaikan, karena hanya AccountUUID
//     yang dipakai untuk mengidentifikasi rekening.
//  3. Menyimpan transaksi uang keluar (fromTransactionOrm) untuk pengirim
//     dan transaksi uang masuk (toTransactionOrm) untuk penerima.
//  4. Mengurangi saldo pengirim dan menambah saldo penerima.
//
// Keempat langkah ini bersifat all-or-nothing: jika salah satunya gagal,
// seluruh perubahan di-rollback dan tidak ada data yang tersimpan.
//
// Nilai kembalian:
//   - (true, nil) jika transfer berhasil dan sudah di-commit.
//   - (false, ErrInsufficientBalance) jika saldo pengirim tidak mencukupi.
//   - (false, ErrSameAccountTransfer) jika rekening pengirim dan penerima sama.
//   - (false, err) untuk kegagalan lain, misalnya rekening tidak ditemukan
//     atau kesalahan database.
func (a *DatabaseAdapter) CreateTransferTransactionPair(fromAccountOrm BankAccountOrm, toAccountOrm BankAccountOrm,
	fromTransactionOrm BankTransactionOrm, toTransactionOrm BankTransactionOrm) (bool, error) {

	// Transfer ke rekening sendiri tidak masuk akal dan akan mengunci row yang sama dua kali
	if fromAccountOrm.AccountUUID == toAccountOrm.AccountUUID {
		return false, ErrSameAccountTransfer
	}

	err := a.db.Transaction(func(tx *gorm.DB) error {
		// 1. Lock kedua akun dengan urutan konsisten (anti-deadlock)
		ids := []uuid.UUID{fromAccountOrm.AccountUUID, toAccountOrm.AccountUUID}
		sort.Slice(ids, func(i, j int) bool { return ids[i].String() < ids[j].String() })

		locked := make(map[uuid.UUID]*BankAccountOrm, 2)

		for _, id := range ids {
			var acct BankAccountOrm

			if err := tx.
				Clauses(clause.Locking{Strength: "UPDATE"}).
				Where("account_uuid = ?", id). // sesuaikan dengan primary key kamu
				First(&acct).Error; err != nil {
				return err
			}

			locked[id] = &acct
		}

		from := locked[fromAccountOrm.AccountUUID]
		to := locked[toAccountOrm.AccountUUID]

		// 2. Hitung saldo baru dari saldo yang sudah di-lock
		fromBalanceNew := from.CurrentBalance - fromTransactionOrm.Amount
		if fromBalanceNew < 0 {
			return ErrInsufficientBalance
		}

		toBalanceNew := to.CurrentBalance + toTransactionOrm.Amount

		// 3. Simpan kedua transaksi (pakai pointer)
		if err := tx.Create(&fromTransactionOrm).Error; err != nil {
			return err
		}

		if err := tx.Create(&toTransactionOrm).Error; err != nil {
			return err
		}

		// 4. Update saldo kedua akun
		now := time.Now()

		if err := tx.Model(from).Updates(map[string]any{
			"current_balance": fromBalanceNew,
			"updated_at":      now,
		}).Error; err != nil {
			return err
		}

		return tx.Model(to).Updates(map[string]any{
			"current_balance": toBalanceNew,
			"updated_at":      now,
		}).Error
	})
	if err != nil {
		return false, err
	}

	return true, nil
}

func (a *DatabaseAdapter) UpdateTransferStatus(transfer BankTransferOrm, status bool) error {
	if err := a.db.Model(&transfer).Updates(
		map[string]any{
			"transfer_success": status,
			"updated_at":       time.Now(),
		},
	).Error; err != nil {
		return err
	}

	return nil
}
