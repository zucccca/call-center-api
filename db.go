package main

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

var db *sql.DB

func InitDB(connectionStr string) (*sql.DB, error){
	var err error
	db, err = sql.Open("postgres", connectionStr)

	if err != nil {
		fmt.Printf("Error establishing connection to database %v", err)
		return nil, err
	}
	
	errCheck := db.Ping()

	if errCheck != nil {
		fmt.Printf("Error pinging database %v", errCheck)
		return nil, errCheck
	}

	fmt.Println("Successfully connected to call center api database")

	return db, nil
}
