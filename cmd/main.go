package main

import (
	"database/sql"
	"log"

	dbmigration "github.com/aditya3232/my-grpc-go-server/db"
	mydb "github.com/aditya3232/my-grpc-go-server/internal/adapter/database"
	mygrpc "github.com/aditya3232/my-grpc-go-server/internal/adapter/grpc"
	app "github.com/aditya3232/my-grpc-go-server/internal/application"

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
