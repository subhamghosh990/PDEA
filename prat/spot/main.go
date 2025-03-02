package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

var (
	db *sql.DB
)

func initDb() {
	var err error
	connStr := "user=pdea password=pdea dbname=pdea sslmode=disable port=5433"
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Println("failed to open postgres connection", err)
		log.Fatal(err)
	}

	schema := `
		CREATE SCHEMA IF NOT EXISTS pdea_practice;

		CREATE TABLE IF NOT EXISTS pdea_practice.vehicle_records(
		id SERIAL PRIMARY KEY,
		spot_number TEXT NOT NULL,
		license_plate TEXT NOT NULL,
		entry_time TIMESTAMP NOT NULL,
		exit_time TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS pdea_practice.parking_spots (
		id SERIAL PRIMARY KEY,
		spot_number TEXT NOT NULL,
		type TEXT NOT NULL,
		is_available TEXT NOT NULL
		);
	`
	_, err = db.Exec(schema)
	if err != nil {
		log.Println("failed to create schema", err)
		log.Fatal(err)
	}
}

type ParkingSpot struct {
	ID          int    `json:"id"`
	SpotNumber  string `json:"spot_number"`
	Type        string `json:"type"`
	IsAvailable string `json:"is_available"`
}

func isValidSlotType(slotType string) bool {
	if slotType == "Compact" || slotType == "Standard" || slotType == "Large" {
		return true
	}
	return false
}

func isAvailableValid(isAvail string) bool {
	if isAvail == "yes" || isAvail == "no" {
		return true
	}
	return false
}

func AddParkingSpot(w http.ResponseWriter, r *http.Request) {
	var parkingSpot ParkingSpot
	if err := json.NewDecoder(r.Body).Decode(&parkingSpot); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}
	if !isValidSlotType(parkingSpot.Type) {
		http.Error(w, "Invalid type. Must be one of: Compact, Standard, Large.", http.StatusBadRequest)
		return
	}
	if !isAvailableValid(parkingSpot.IsAvailable) {
		http.Error(w, "Invalid available type. Must be yes or no", http.StatusBadRequest)
		return
	}
	parkingSpots, err := getAllParkingSpots()
	if err != nil {
		http.Error(w, "Invalid available type. Must be yes or no", http.StatusBadRequest)
		return
	}
	var serial_id int
	for _, spot := range parkingSpots {
		if spot.SpotNumber == parkingSpot.SpotNumber {
			http.Error(w, "Parking Spot already added", http.StatusBadRequest)
			return
		}
		if serial_id < spot.ID {
			serial_id = spot.ID
		}
	}
	serial_id++
	parkingSpot.ID = serial_id
	insertQuery := `INSERT into pdea_practice.parking_spots(id,spot_number,type,is_available) values ($1,$2,$3,$4)`
	_, err = db.Exec(insertQuery, parkingSpot.ID, parkingSpot.SpotNumber, parkingSpot.Type, parkingSpot.IsAvailable)
	if err != nil {
		log.Println("failed to insert parking spot details to db ", err)
		http.Error(w, "Failed to add parking spot to db", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	res, err := json.Marshal(parkingSpot)
	if err != nil {
		log.Println("failed to marshal to json ", err)
		return
	}
	w.Write(res)
}

func getAllParkingSpots() ([]ParkingSpot, error) {
	var parkingSpots []ParkingSpot
	query := `SELECT id, spot_number, type, is_available from pdea_practice.parking_spots`
	rows, err := db.Query(query)
	if err != nil {
		log.Println("Error executing db query for get all parking spots ", err)
		return nil, err
	}
	for rows.Next() {
		var parkingSpot ParkingSpot
		err := rows.Scan(&parkingSpot.ID, &parkingSpot.SpotNumber, &parkingSpot.Type, &parkingSpot.IsAvailable)
		if err != nil {
			log.Println("Error scanning db rows ", err)
			continue
		}
		parkingSpots = append(parkingSpots, parkingSpot)
	}
	return parkingSpots, err
}

func GetAllParkingSpots(w http.ResponseWriter, r *http.Request) {
	parkingSpots, err := getAllParkingSpots()
	if err != nil {
		log.Println("failed to get parking spots from db ", err)
		http.Error(w, "failed to get parking spots", http.StatusInternalServerError)
		return
	}
	res, err := json.Marshal(parkingSpots)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to json marhsal output object. error: %v", err), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(res)
}

func GetParkingSpot(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	requestedId := vars["id"]
	reqId, err := strconv.Atoi(requestedId)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid spot id: %v", err), http.StatusBadRequest)
		return
	}
	parkingSpots, err := getAllParkingSpots()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to parking spot. error: %v", err), http.StatusInternalServerError)
		return
	}
	var parkingSpot ParkingSpot
	var found bool
	for _, pSpot := range parkingSpots {
		if pSpot.ID == reqId {
			parkingSpot = pSpot
			found = true
			break
		}
	}
	if !found {
		http.Error(w, "Parking spot not found", http.StatusNotFound)
		return
	}
	res, err := json.Marshal(parkingSpot)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to marshal to json data. error: %v", err), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusFound)
	w.Write(res)
}

func UpdateParkingSpot(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	stringId := vars["id"]
	intId, err := strconv.Atoi(stringId)
	if err != nil {
		http.Error(w, "Invalid id", http.StatusBadRequest)
		return
	}
	var foundParkingSpot, parkingSpot ParkingSpot
	if err := json.NewDecoder(r.Body).Decode(&parkingSpot); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}
	if !isValidSlotType(parkingSpot.Type) {
		http.Error(w, "Invalid type. Must be one of: Compact, Standard, Large.", http.StatusBadRequest)
		return
	}
	if !isAvailableValid(parkingSpot.IsAvailable) {
		http.Error(w, "Invalid available type. Must be yes or no", http.StatusBadRequest)
		return
	}
	parkingSpots, err := getAllParkingSpots()
	if err != nil {
		http.Error(w, "Invalid available type. Must be yes or no", http.StatusBadRequest)
		return
	}
	var found bool
	for _, spot := range parkingSpots {
		if spot.ID == intId {
			foundParkingSpot = spot
			found = true
			break
		}
	}
	if !found {
		http.Error(w, "Parking spot not found", http.StatusNotFound)
		return
	}
	foundParkingSpot.SpotNumber = parkingSpot.SpotNumber
	foundParkingSpot.Type = parkingSpot.Type
	foundParkingSpot.IsAvailable = parkingSpot.IsAvailable
	updateQuery := `UPDATE pdea_practice.parking_spots set spot_number = $1 ,type = $2 ,is_available = $3 where id=$4`
	_, err = db.Exec(updateQuery, foundParkingSpot.SpotNumber, foundParkingSpot.Type, foundParkingSpot.IsAvailable, foundParkingSpot.ID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to update parking spot. Error: %v", err), http.StatusInternalServerError)
		return
	}
	res, err := json.Marshal(foundParkingSpot)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to marshal to json object. Error: %v", err), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusAccepted)
	w.Write(res)
}

func DeleteParkingSpot(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	stringId := vars["id"]
	intId, err := strconv.Atoi(stringId)
	if err != nil {
		http.Error(w, "Invalid id", http.StatusBadRequest)
		return
	}
	parkingSpots, err := getAllParkingSpots()
	if err != nil {
		http.Error(w, "Invalid available type. Must be yes or no", http.StatusBadRequest)
		return
	}
	var found bool
	for _, spot := range parkingSpots {
		if spot.ID == intId {
			found = true
			break
		}
	}
	if !found {
		http.Error(w, "Parking spot not found", http.StatusNotFound)
		return
	}
	query := `DELETE FROM pdea_practice.parking_spots where id = $1`
	_, err = db.Exec(query, intId)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to delete parking spot. Error %v", err), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	type Response struct {
		Message string
	}
	response := Response{
		Message: "Parking spot has been deleted succesfully",
	}
	res, err := json.Marshal(response)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to marshal response object. Error %v", err), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(res)

}

func main() {
	initDb()
	registerRoutes()
}

func registerRoutes() {
	router := mux.NewRouter()
	router.HandleFunc("/api/parking-spots", AddParkingSpot).Methods("POST")
	router.HandleFunc("/api/parking-spots/all", GetAllParkingSpots).Methods("GET")
	router.HandleFunc("/api/parking-spots/{id}", GetParkingSpot).Methods("GET")
	router.HandleFunc("/api/parking-spots/{id}", UpdateParkingSpot).Methods("PUT")
	router.HandleFunc("/api/parking-spots/{id}", DeleteParkingSpot).Methods("DELETE")
	log.Println("ParkingSpot app started on port 8080")
	http.ListenAndServe("localhost:8080", router)
	log.Println("ParkingSpot app stopped on port 8080")
}
