package main

import (
	"database/sql"
	"log"
	"math/rand"
	"time"

	dbmigration "github.com/aditya3232/my-grpc-go-server/db"
	mydb "github.com/aditya3232/my-grpc-go-server/internal/adapter/database"
	mygrpc "github.com/aditya3232/my-grpc-go-server/internal/adapter/grpc"
	app "github.com/aditya3232/my-grpc-go-server/internal/application"
	"github.com/aditya3232/my-grpc-go-server/internal/application/domain/bank"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	log.SetFlags(0)
	log.SetOutput(logWriter{})

	sqlDB, err := sql.Open("pgx", "postgres://root:root_password@localhost:5432/grpc?sslmode=disable")
	if err != nil {
		log.Fatalln("Can't connect to database : ", err)
	}

	dbmigration.Migrate(sqlDB)

	databaseAdapter, err := mydb.NewDatabaseAdapter(sqlDB)
	if err != nil {
		log.Fatalln("Can't create database adapter : ", err)
	}

	hs := &app.HelloService{}
	bs := app.NewBankService(databaseAdapter)

	go generateExchangeRates(bs, "USD", "IDR", 5*time.Second)

	grpcAdapter := mygrpc.NewGrpcAdapter(hs, bs, 9090)

	grpcAdapter.Run()
}

// func runDummyOrm(da *mydb.DatabaseAdapter) {
// 	now := time.Now()

// 	uuid, _ := da.Save(
// 		&mydb.DummyOrm{
// 			UserID:    uuid.New(),
// 			UserName:  "Tim " + time.Now().Format("15:04:05"),
// 			CreatedAt: now,
// 			UpdatedAt: now,
// 		},
// 	)

// 	res, _ := da.GetByUUID(&uuid)

// 	log.Println("res : ", res)
// }

func generateExchangeRates(bs *app.BankService, fromCurrency, ToCurrency string, duration time.Duration) {
	ticker := time.NewTicker(duration)

	for range ticker.C {
		now := time.Now()
		validFrom := now.Truncate(time.Second).Add(3 * time.Second)
		validTo := validFrom.Add(duration).Add(-1 * time.Millisecond)

		dummyrate := bank.ExchangeRate{
			FromCurrency:       fromCurrency,
			ToCurrency:         ToCurrency,
			Rate:               2000 + float64(rand.Intn(300)),
			ValidFromTimestamp: validFrom,
			ValidToTimestamp:   validTo,
		}

		bs.CreateExchangeRate(dummyrate)
	}
}
