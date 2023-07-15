package database

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
)

const (
	host     = "localhost"
	port     = 5432
	user     = "yourusername"
	password = "yourpassword"
	dbname   = "yourdatabase"
)

var db *sql.DB

func init() {
	var err error
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)
	db, err = sql.Open("postgres", psqlInfo)
	if err != nil {
		panic(err)
	}
	err = db.Ping()
	if err != nil {
		panic(err)
	}
	log.Println("Successfully connected!")
}

func LogRequest(requestInfo string) {
	sqlStatement := `INSERT INTO requests (info) VALUES ($1)`
	_, err := db.Exec(sqlStatement, requestInfo)
	if err != nil {
		log.Println(err)
	}
}
