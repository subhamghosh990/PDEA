package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

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

type VehicleRecord struct {
	ID           int    `json:"id"`
	SpotNumber   string `json:"spot_number"`
	LicensePlate string `json:"license_plate"`
	EntryTime    string `json:"entry_time,omitempty"`
	ExitTime     string `json:"exit_time,omitempty"`
}

type VehicleRecordEntry struct {
	ID           int       `json:"id"`
	SpotNumber   string    `json:"spot_number"`
	LicensePlate string    `json:"license_plate"`
	EntryTime    time.Time `json:"entry_time"`
	ExitTime     time.Time `json:"exit_time"`
}

func main() {
	initDb()
	registerRoutes()
}

func formatDateTime(t time.Time) string {
	return t.Format("01-02-2006 15:04:05")
}

func VehicleEntry(w http.ResponseWriter, r *http.Request) {
	var parkingEntry VehicleRecord
	if err := json.NewDecoder(r.Body).Decode(&parkingEntry); err != nil {
		log.Printf("Failed to decode request body, error: %v time: %v", err, formatDateTime(time.Now()))
		http.Error(w, "Failed to decode request body", http.StatusBadRequest)
		return
	}
	vehicleEntries, err := getAllVehicleEntry()
	if err != nil {
		log.Printf("Failed to check vehicle entries, error: %v time: %v", err, formatDateTime(time.Now()))
		http.Error(w, "Failed to check vehicle entries", http.StatusInternalServerError)
		return
	}
	var id int
	for _, entry := range vehicleEntries {
		if parkingEntry.LicensePlate == entry.LicensePlate && entry.ExitTime.IsZero() {
			log.Printf("Vehicle already parked time: %v", formatDateTime(time.Now()))
			http.Error(w, "Vehicle already parked", http.StatusBadRequest)
			return
		}
		id++
	}
	parkingSpot, err := getParkingSpotBySpotNumber(parkingEntry.SpotNumber)
	if err != nil {
		log.Printf("Error in getting parking spot. Error:%v Time: %v", err, formatDateTime(time.Now()))
		http.Error(w, "Parking Spot Not Found", http.StatusNotFound)
		return
	}
	if parkingSpot.SpotNumber == parkingEntry.SpotNumber {
		if parkingSpot.IsAvailable != "yes" {
			log.Printf("Parking spot already occupied time: %v ", formatDateTime(time.Now()))
			http.Error(w, "Parking spot already occupied", http.StatusInternalServerError)
			return
		}
	}
	vehicleRecordEntry := VehicleRecordEntry{}
	vehicleRecordEntry.LicensePlate = parkingEntry.LicensePlate
	vehicleRecordEntry.SpotNumber = parkingEntry.SpotNumber
	vehicleRecordEntry.EntryTime = time.Now()
	id++
	vehicleRecordEntry.ID = id
	err = insertVehicleRecord(&vehicleRecordEntry)
	if err != nil {
		log.Printf("Error inserting vehicle record. Error: %v time: %v id %v", err, formatDateTime(time.Now()), id)
		http.Error(w, "Failed to insert vehicle record", http.StatusInternalServerError)
		return
	}
	parkingEntry.ID = int(id)
	parkingEntry.EntryTime = formatDateTime(vehicleRecordEntry.EntryTime)
	parkingSpot.IsAvailable = "no"
	err = updateParkingSpot(parkingSpot)
	if err != nil {
		log.Printf("Error updating parking spot. Error: %v time: %v", err, formatDateTime(time.Now()))
		http.Error(w, "Failed to update parking spot", http.StatusInternalServerError)
		return
	}
	output, err := json.Marshal(parkingEntry)
	if err != nil {
		log.Printf("Error framing response. Error: %v time: %v ", err, formatDateTime(time.Now()))
		http.Error(w, "Error framing response", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(output)
}

func updateParkingSpot(parkingSpot *ParkingSpot) error {
	query := `UPDATE pdea_practice.parking_spots set is_available=$1 where id=$2`
	_, err := db.Exec(query, parkingSpot.IsAvailable, parkingSpot.ID)
	if err != nil {
		log.Printf("Error updating parking spot. Error: %v time: %v ", err, formatDateTime(time.Now()))
		return err
	}
	return nil
}

func getParkingSpotBySpotNumber(spotNumber string) (*ParkingSpot, error) {
	query := `SELECT id, spot_number,type,is_available from pdea_practice.parking_spots where spot_number=$1`
	rows, err := db.Query(query, spotNumber)
	if err != nil {
		log.Printf("Failed to get parking spot from db, Error %v Spot Number %s", err, spotNumber)
		return nil, err
	}
	for rows.Next() {
		var parkingSpot ParkingSpot
		err = rows.Scan(&parkingSpot.ID, &parkingSpot.SpotNumber, &parkingSpot.Type, &parkingSpot.IsAvailable)
		if err != nil {
			log.Printf("Failed to scan parking spot from db, Error %v Spot Number %s", err, spotNumber)
			return nil, err
		}
		return &parkingSpot, nil
	}
	return nil, errors.New("parking spot not found")
}

func insertVehicleRecord(parkingEntry *VehicleRecordEntry) error {
	query := `INSERT INTO pdea_practice.vehicle_records (id,spot_number,license_plate,entry_time) values ($1,$2,$3,$4)`
	_, err := db.Exec(query, parkingEntry.ID, parkingEntry.SpotNumber, parkingEntry.LicensePlate, parkingEntry.EntryTime)
	if err != nil {
		log.Printf("Error running insert query. Error: %v time: %v request time %v", err, formatDateTime(time.Now()), parkingEntry.EntryTime)
		return err
	}
	return err
}

func updateVehicleRecord(parkingEntry *VehicleRecordEntry) error {
	query := `UPDATE pdea_practice.vehicle_records SET exit_time =$1 where id = $2`
	_, err := db.Exec(query, parkingEntry.ExitTime, parkingEntry.ID)
	if err != nil {
		log.Printf("Error running insert query. Error: %v time: %v request time %v", err, formatDateTime(time.Now()), parkingEntry.EntryTime)
		return err
	}
	return err
}

func getAllVehicleEntry() ([]VehicleRecordEntry, any) {
	query := `select id, spot_number,license_plate,entry_time,exit_time from pdea_practice.vehicle_records`
	rows, err := db.Query(query)
	if err != nil {
		log.Printf("Error in querying db. Error: %v time: %v", err, formatDateTime(time.Now()))
		return nil, err
	}
	var vehicleRecords []VehicleRecordEntry
	for rows.Next() {
		var vehicleRecord VehicleRecordEntry
		var nullableTime sql.NullTime
		err = rows.Scan(&vehicleRecord.ID, &vehicleRecord.SpotNumber, &vehicleRecord.LicensePlate, &vehicleRecord.EntryTime, &nullableTime)
		if err != nil {
			log.Printf("Error in scanning vehicle record. Error: %v time: %v", err, formatDateTime(time.Now()))
			continue
		}
		if nullableTime.Valid {
			vehicleRecord.ExitTime = nullableTime.Time
		}
		vehicleRecords = append(vehicleRecords, vehicleRecord)
	}
	return vehicleRecords, nil
}

func VehicleExit(w http.ResponseWriter, r *http.Request) {
	var parkingEntry VehicleRecord
	if err := json.NewDecoder(r.Body).Decode(&parkingEntry); err != nil {
		log.Printf("Failed to decode request body, error: %v time: %v", err, formatDateTime(time.Now()))
		http.Error(w, "Failed to decode request body", http.StatusBadRequest)
		return
	}
	vehicleEntries, err := getAllVehicleEntry()
	if err != nil {
		log.Printf("Failed to check vehicle entries, error: %v time: %v", err, formatDateTime(time.Now()))
		http.Error(w, "Failed to check vehicle entries", http.StatusInternalServerError)
		return
	}
	var vehicleEntry VehicleRecordEntry
	var foundEntry bool
	for _, entry := range vehicleEntries {
		if parkingEntry.LicensePlate == entry.LicensePlate && parkingEntry.SpotNumber == entry.SpotNumber {
			if !entry.ExitTime.IsZero() {
				log.Printf("Vehicle already exited: %v", formatDateTime(time.Now()))
			} else {
				vehicleEntry = entry
				foundEntry = true
			}
		}
	}
	if !foundEntry {
		log.Printf("No vehicle parked for this number, vehicle: %v time: %v", parkingEntry, formatDateTime(time.Now()))
		http.Error(w, "No vehicle parked for this number at given spot", http.StatusInternalServerError)
		return
	}
	parkingSpot, err := getParkingSpotBySpotNumber(vehicleEntry.SpotNumber)
	if err != nil {
		log.Printf("Error finding parking spot, Error: %v time: %v", err, formatDateTime(time.Now()))
		http.Error(w, "Parking Spot Not Found", http.StatusNotFound)
		return
	}
	if parkingSpot.SpotNumber == parkingEntry.SpotNumber {
		if parkingSpot.IsAvailable != "no" {
			log.Printf("Parking spot is already released time: %v ", formatDateTime(time.Now()))
			http.Error(w, "Parking spot is already released", http.StatusInternalServerError)
			return
		}
	}
	vehicleRecordEntry := VehicleRecordEntry{}
	vehicleRecordEntry.ID = vehicleEntry.ID
	vehicleRecordEntry.LicensePlate = parkingEntry.LicensePlate
	vehicleRecordEntry.SpotNumber = parkingEntry.SpotNumber
	vehicleRecordEntry.EntryTime = vehicleEntry.EntryTime
	vehicleRecordEntry.ExitTime = time.Now()
	err = updateVehicleRecord(&vehicleRecordEntry)
	if err != nil {
		log.Printf("Error updating vehicle record. Error: %v time: %v ", err, formatDateTime(time.Now()))
		http.Error(w, "Failed to update vehicle record", http.StatusInternalServerError)
		return
	}
	parkingSpot.IsAvailable = "yes"
	err = updateParkingSpot(parkingSpot)
	if err != nil {
		log.Printf("Failed to update parking spot. Error: %v time: %v ", err, formatDateTime(time.Now()))
		http.Error(w, "Failed to update parking spot", http.StatusInternalServerError)
		return
	}
	parkingEntry.ID = vehicleRecordEntry.ID
	parkingEntry.EntryTime = formatDateTime(vehicleRecordEntry.EntryTime)
	parkingEntry.ExitTime = formatDateTime(vehicleRecordEntry.ExitTime)
	output, err := json.Marshal(parkingEntry)
	if err != nil {
		log.Printf("Error framing response. Error: %v time: %v ", err, formatDateTime(time.Now()))
		http.Error(w, "Error framing response", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(output)
}

func registerRoutes() {
	router := mux.NewRouter()
	router.HandleFunc("/api/vehicle-entries", VehicleEntry).Methods("POST")
	router.HandleFunc("/api/vehicle-exits", VehicleExit).Methods("POST")
	log.Println("VehicleEntryExit app started on port 8081")
	http.ListenAndServe("localhost:8081", router)
	log.Println("VehicleEntryExit app stopped on port 8081")
}
