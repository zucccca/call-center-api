package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	_ "github.com/lib/pq"
)

var db *sql.DB

func InitDB(connectionStr string) (*sql.DB, error) {
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
		return -1, err
	}

	err = db.QueryRow(
		`INSERT INTO calls(filename, transcript, flags, flag_count, is_pushy, score, agent_name, trackdrive_url, disposition, offer_name, agent_talk_time, forward_duration) 
		 VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12) RETURNING id`,
		callData.Filename,
		callData.Transcript,
		flags,
		callData.FlagCount,
		callData.IsPushy,
		callData.Score,
		callData.AgentName,
		callData.TrackdriveUrl,
		callData.Disposition,
		callData.OfferName,
		callData.AgentTalkTime,
		callData.ForwardDuration,
	).Scan(&callId)

	if err != nil {
		log.Printf("Failed saving call data associated with filename -> %s, err %v", callData.Filename, err)
		return -1, err
	}

	return callId, nil
}

func GetCalls(limit, offset int, filters CallFilters) ([]CallSummary, int, error) {

	// Build dynamic WHERE clause
	conditions := []string{}
	args := []interface{}{}
	argIndex := 1

	if filters.AgentName != "" {
		conditions = append(conditions, fmt.Sprintf("LOWER(agent_name) LIKE LOWER($%d)", argIndex))
		args = append(args, "%"+filters.AgentName+"%")
		argIndex++
	}

	if filters.Disposition != "" {
		conditions = append(conditions, fmt.Sprintf("disposition = $%d", argIndex))
		args = append(args, filters.Disposition)
		argIndex++
	}

	if filters.DateFrom != "" {
		conditions = append(conditions, fmt.Sprintf("created_at >= $%d", argIndex))
		args = append(args, filters.DateFrom)
		argIndex++
	}

	if filters.DateTo != "" {
		conditions = append(conditions, fmt.Sprintf("created_at <= $%d", argIndex))
		args = append(args, filters.DateTo)
		argIndex++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Sorting
	sortBy := "created_at"
	if filters.SortBy == "score" || filters.SortBy == "flag_count" {
		sortBy = filters.SortBy
	}

	sortOrder := "DESC"
	if strings.ToUpper(filters.SortOrder) == "ASC" {
		sortOrder = "ASC"
	}

	// Count query
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM calls %s", whereClause)
	var total int
	err := db.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		log.Printf("Failed counting calls, err %v", err)
		return nil, 0, err
	}

	// Add limit and offset args
	args = append(args, limit, offset)

	// Main query
	query := fmt.Sprintf(`
		SELECT id, filename, score, flag_count, is_pushy, created_at, agent_name, trackdrive_url, disposition, offer_name
		FROM calls
		%s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d
	`, whereClause, sortBy, sortOrder, argIndex, argIndex+1)

	rows, err := db.Query(query, args...)
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
			&call.AgentName,
			&call.TrackdriveUrl,
			&call.Disposition,
			&call.OfferName,
		)
		if err != nil {
			log.Printf("Failed scanning row, err %v", err)
			return nil, 0, err
		}
		calls = append(calls, call)
	}

	return calls, total, nil
}

func GetCall(id int) (*CallDetail, error) {
	var call CallDetail
	var flagsJSON []byte
	err := db.QueryRow(
		`SELECT id, filename, transcript, flags, flag_count, is_pushy, score, created_at, agent_name, trackdrive_url, disposition, offer_name, agent_talk_time, forward_duration 
		 FROM calls WHERE id = $1`, id,
	).Scan(
		&call.ID,
		&call.Filename,
		&call.Transcript,
		&flagsJSON,
		&call.FlagCount,
		&call.IsPushy,
		&call.Score,
		&call.CreatedAt,
		&call.AgentName,
		&call.TrackdriveUrl,
		&call.Disposition,
		&call.OfferName,
		&call.AgentTalkTime,
		&call.ForwardDuration,
	)

	if err != nil {
		log.Printf("Failed fetching call with id %d, err %v", id, err)
		return nil, err
	}

	err = json.Unmarshal(flagsJSON, &call.Flags)
	if err != nil {
		log.Printf("Failed unmarshaling flags, err %v", err)
		return nil, err
	}

	return &call, nil
}
