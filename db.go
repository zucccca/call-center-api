package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

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

func SaveCall(callData *CallCompliance) (int, error) {
	var callId int
	flags, err := json.Marshal(callData.Flags)

	if err != nil {
		log.Printf("Error encoding call flags %v", err)
		return  -1, err
	}
	
	err = db.QueryRow(`INSERT INTO calls(filename, transcript, flags, flag_count, is_pushy, score) VALUES($1, $2, $3, $4, $5, $6) RETURNING id`, 
	callData.Filename, callData.Transcript, flags, callData.FlagCount, callData.IsPushy, callData.Score).Scan(&callId)

	if err != nil {
		log.Printf("Failed saving call data associated with filename -> %s, err %v", callData.Filename, err)
		return -1, err
	}

	return callId, nil
}

func GetCalls(limit, offset int) ([]CallSummary, int, error) {
  
    var total int
    err := db.QueryRow("SELECT COUNT(*) FROM calls").Scan(&total)
    if err != nil {
        log.Printf("Failed counting calls, err %v", err)
        return nil, 0, err
    }

    rows, err := db.Query(`
        SELECT id, filename, score, flag_count, is_pushy, created_at 
        FROM calls 
        ORDER BY created_at DESC 
        LIMIT $1 OFFSET $2
    `, limit, offset)
    
    if err != nil {
        log.Printf("Failed fetching calls from db, err %v", err)
        return nil, 0, err
    }
    defer rows.Close()

    calls := []CallSummary{}
    for rows.Next() {
        var call CallSummary
        err := rows.Scan(
            &call.ID,
            &call.Filename,
            &call.Score,
            &call.FlagCount,
            &call.IsPushy,
            &call.CreatedAt,
        )
        if err != nil {
            log.Printf("Failed scanning row, err %v", err)
            return nil, 0, err
        }
        calls = append(calls, call)
    }

    return calls, total, nil
}
