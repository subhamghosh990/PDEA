package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

type ParkingSpot struct {
	ID          int    `json:"id"`
	SpotNumber  string `json:"spot_number"`
	Type        string `json:"type"`
	IsAvailable bool   `json:"is_available"`
}

type Vehichle struct {
	ID            int       `json:"id"`
	SpotNumber    string    `json:"spot_number"`
	License_plate string    `json:"license_plate"`
	EntryTime     time.Time `json:"entry_time"`
	ExitTime      time.Time `json:"exit_time"`
}

var (
	db *sql.DB
)
var static_id int

func connectDB() {
	// Define the connection string
	connStr := "host=10.24.113.223 port=5432 user=robot password=cisco123 dbname=pdea sslmode=disable"
	var err error
	// Open a connection to the database
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		fmt.Printf("Error opening database: %v", err)
	}

	// Ping the database to verify the connection
	err = db.Ping()
	if err != nil {
		fmt.Printf("Error connecting to the database: %v", err)
	}

	fmt.Println("Successfully connected to the PostgreSQL database!")

	// Define the SQL to create a schema
	schemaSQL := `
CREATE TABLE IF NOT EXISTS vehicle_records (
id SERIAL PRIMARY KEY,
spot_number TEXT NOT NULL,
license_plate TEXT NOT NULL,
entry_time TIMESTAMP NOT NULL,
exit_time TIMESTAMP
);

CREATE TABLE IF NOT EXISTS parking_spots (
id SERIAL PRIMARY KEY,
spot_number TEXT ,
type TEXT NOT NULL,
is_available  BOOLEAN NOT NULL
);

CREATE TABLE IF NOT EXISTS parking_rec (
spot_number TEXT PRIMARY KEY,
type TEXT NOT NULL,
is_available  BOOLEAN NOT NULL
);
`

	// Execute the SQL statement
	_, err = db.Exec(schemaSQL)
	if err != nil {
		fmt.Printf("Error creating schema: %v", err)
	}

	fmt.Println("Schema 'my_schema' created successfully!")
}

func getCarsDataAll() ([]Vehichle, error) {
	qr := `select id,spot_number,license_plate, entry_time from vehicle_records;`
	var car Vehichle
	var res []Vehichle

	rows, err := db.Query(qr)
	if err != nil {
		return res, err
	}
	defer rows.Close()
	for rows.Next() {
		err := rows.Scan(&car.ID, &car.SpotNumber, &car.License_plate, &car.EntryTime)
		if err != nil {
			continue
		}
		res = append(res, car)
	}
	return res, nil
}

func getCarsDataAllValue() ([]Vehichle, error) {
	qr := `select id,spot_number,license_plate, entry_time, exit_time from vehicle_records;`
	var res []Vehichle

	rows, err := db.Query(qr)
	if err != nil {
		return res, err
	}
	defer rows.Close()
	for rows.Next() {
		var car Vehichle
		err := rows.Scan(&car.ID, &car.SpotNumber, &car.License_plate, &car.EntryTime, &car.ExitTime)
		if err != nil {
			fmt.Println("subham err", err)
			//continue
		}
		res = append(res, car)
	}
	return res, nil
}
func getParkingSpotData(id string) (ParkingSpot, error) {
	qr := `select spot_number, type ,is_available from  parking_rec where spot_number = $1;`
	row := db.QueryRow(qr, id)
	var res ParkingSpot
	err := row.Scan(&res.SpotNumber, &res.Type, &res.IsAvailable)
	return res, err

}

func updateParkingSpotData(p ParkingSpot) error {
	qr := `UPDATE parking_rec SET is_available = $1 where spot_number = $2;`
	_, err := db.Exec(qr, p.IsAvailable, p.SpotNumber)
	return err
}

func insertCarData(v Vehichle) error {
	qr := `INSERT INTO vehicle_records (id , spot_number, license_plate , entry_time) VALUES($1,$2,$3,$4) ;`
	_, err := db.Exec(qr, v.ID, v.SpotNumber, v.License_plate, v.EntryTime)
	return err
}

func updateCarData(v Vehichle) error {
	qr := `UPDATE vehicle_records SET exit_time = $1 where spot_number = $2 and license_plate = $3;`
	_, err := db.Exec(qr, v.ExitTime, v.SpotNumber, v.License_plate)
	return err
}
func formatTime(t time.Time) string {
	return t.Format("02-01-2006 15:04:05")
}
func RegisterEntry(w http.ResponseWriter, r *http.Request) {
	fmt.Println("RegisterEntry")
	var reqBody Vehichle
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		http.Error(w, "Inavlid req body", http.StatusBadRequest)
		return
	}
	p, err := getParkingSpotData(reqBody.SpotNumber)
	if err != nil && errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "Parking spot not found", http.StatusNotFound)
		fmt.Println("RegisterEntry 1 err - ", err)
		return
	}
	if !p.IsAvailable {
		http.Error(w, "Parking spot not available", http.StatusNotFound)
		return
	}
	cars, err := getCarsDataAll()
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		fmt.Println("RegisterEntry 2 err - ", err)
		return
	}
	var static_id int
	for _, d := range cars {
		if static_id <= d.ID {
			static_id = d.ID
		}
	}
	static_id++
	reqBody.ID = static_id
	reqBody.EntryTime = time.Now()
	p.IsAvailable = false
	err = insertCarData(reqBody)
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		fmt.Println("RegisterEntry 4 err - ", err)
		return
	}
	err = updateParkingSpotData(p)
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		fmt.Println("RegisterEntry 3 err - ", err)
		return
	}
	resJson, _ := json.Marshal(reqBody)
	w.WriteHeader(http.StatusCreated)
	w.Write(resJson)
}
func RegisterExit(w http.ResponseWriter, r *http.Request) {
	fmt.Println("RegisterExit")
	var reqBody Vehichle
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		http.Error(w, "Inavlid req body", http.StatusBadRequest)
		return
	}
	p, err := getParkingSpotData(reqBody.SpotNumber)
	if err != nil && errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "Parking spot not found", http.StatusNotFound)
		fmt.Println("RegisterExit 1 err - ", err)
		return
	}
	cars, err := getCarsDataAll()
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		fmt.Println("RegisterExit 2 err - ", err)
		return
	}
	var carData Vehichle
	found := false
	for _, c := range cars {
		if c.SpotNumber == reqBody.SpotNumber && c.License_plate == reqBody.License_plate {
			found = true
			carData = c
		}
	}
	if !found {
		http.Error(w, "Vechile Data not found", http.StatusNotFound)
		return
	}
	carData.ExitTime = time.Now()
	err = updateCarData(carData)
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		fmt.Println("RegisterExit 3 err - ", err)
		return
	}
	p.IsAvailable = true
	err = updateParkingSpotData(p)
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		fmt.Println("RegisterExit 4 err - ", err)
		return
	}
	resJson, _ := json.Marshal(carData)
	w.WriteHeader(http.StatusOK)
	w.Write(resJson)
}
func GetVRecordsBySpotNo(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/api/vehicle-records/"):]
	if id == "" {
		http.Error(w, "Server error", http.StatusBadRequest)
		return
	}
	cars, err := getCarsDataAllValue()
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		fmt.Println("RegisterExit 2 err - ", err)
		return
	}
	var res []Vehichle
	fmt.Println(cars)
	for _, c := range cars {
		if c.SpotNumber == id {
			res = append(res, c)
		}
	}
	resJson, _ := json.Marshal(res)
	w.WriteHeader(http.StatusOK)
	w.Write(resJson)
}

// ###################
func insertParkingSpot(d ParkingSpot) error {
	qr := `INSERT INTO parking_spots (id, spot_number, type, is_available) VALUES ($1, $2, $3, $4)`
	_, err := db.Exec(qr, d.ID, d.SpotNumber, d.Type, d.IsAvailable)
	return err
}
func ParkingSpotsEntry(w http.ResponseWriter, r *http.Request) {
	fmt.Println("ParkingSpotsEntry")
	var reqBody ParkingSpot
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	spots, err := getAllParkingSpots()
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		fmt.Println("err - ", err)
		return
	}
	for _, d := range spots {
		if static_id < d.ID {
			static_id = d.ID
		}
	}
	static_id++
	reqBody.ID = static_id
	err = insertParkingSpot(reqBody)
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		fmt.Println("err - ", err)
		return
	}
	jsonRes, _ := json.Marshal(reqBody)
	w.WriteHeader(http.StatusCreated)
	w.Write(jsonRes)
}

func getAllParkingSpots() ([]ParkingSpot, error) {
	fmt.Println("getAllParkingSpots")
	qr := `select id, spot_number, type, is_available from parking_spots;`
	var row ParkingSpot
	var res []ParkingSpot
	rows, err := db.Query(qr)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		err := rows.Scan(&row.ID, &row.SpotNumber, &row.Type, &row.IsAvailable)
		if err != nil {
			fmt.Println("getAllParkingSpots err :", err)
		}
		res = append(res, row)
	}
	return res, nil

}
func ParkingSpotsGetAll(w http.ResponseWriter, r *http.Request) {
	fmt.Println("ParkingSpotsGetAll")
	spots, err := getAllParkingSpots()
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		fmt.Println("err - ", err)
		return
	}
	jsonRes, _ := json.Marshal(spots)
	w.WriteHeader(http.StatusFound)
	w.Write(jsonRes)
}

func ParkingSpotsGetById(w http.ResponseWriter, r *http.Request) {
	fmt.Println("ParkingSpotsGetById")
	id := r.URL.Path[len("/api/parking-spots/"):]
	if id == "" {
		http.Error(w, "Server error", http.StatusBadRequest)
		fmt.Println("ParkingSpotsGetById er:", id)
		return
	}
	idVal, _ := strconv.Atoi(id)
	spots, err := getAllParkingSpots()
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		fmt.Println("ParkingSpotsGetById er:", err)
		return
	}

	for _, data := range spots {
		if data.ID == idVal {
			jsonRes, _ := json.Marshal(data)
			w.WriteHeader(http.StatusFound)
			w.Write(jsonRes)
			return
		}
	}
	http.Error(w, "ID not found", http.StatusNotFound)

}

func updateParkingSpotsData(id int, d ParkingSpot) error {
	qr := `update parking_spots set type = $1 , is_available = $2 where id = $3;`
	_, err := db.Exec(qr, d.Type, d.IsAvailable, d.ID)
	return err
}
func ParkingSpotsUpdate(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/api/parking-spots/"):]
	if id == "" {
		http.Error(w, "Server error", http.StatusBadRequest)
		return
	}
	idVal, _ := strconv.Atoi(id)
	spots, err := getAllParkingSpots()
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}
	found := false
	var res, reqBody ParkingSpot
	for _, data := range spots {
		if data.ID == idVal {
			res = data
			found = true
			break
		}
	}
	if !found {
		http.Error(w, "ID not found", http.StatusNotFound)
		return
	}

	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	res.IsAvailable = reqBody.IsAvailable
	res.Type = reqBody.Type
	err = updateParkingSpotsData(res.ID, res)
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}
	jsonRes, _ := json.Marshal(res)
	w.WriteHeader(http.StatusAccepted)
	w.Write(jsonRes)

}
func deleteParkingSpots(id int) error {
	qr := `delete from parking_spots where id = $1 ;`
	_, err := db.Exec(qr, id)
	return err
}
func ParkingSpotsDelete(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/api/parking-spots/"):]
	if id == "" {
		http.Error(w, "Server error", http.StatusBadRequest)
		return
	}
	idVal, _ := strconv.Atoi(id)
	spots, err := getAllParkingSpots()
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}
	found := false
	for _, data := range spots {
		if data.ID == idVal {
			found = true
			break
		}
	}
	if !found {
		http.Error(w, "ID not found", http.StatusNotFound)
		return
	}
	err = deleteParkingSpots(idVal)
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func registerRoutes() {
	router := mux.NewRouter()
	router.HandleFunc("/api/vehicle-entries", RegisterEntry).Methods("POST")
	router.HandleFunc("/api/vehicle-exits", RegisterExit).Methods("POST")
	router.HandleFunc("/api/vehicle-records/{spot_no}", GetVRecordsBySpotNo).Methods("GET")

	router.HandleFunc("/api/parking-spots", ParkingSpotsEntry).Methods("POST")
	router.HandleFunc("/api/parking-spots/all", ParkingSpotsGetAll).Methods("GET")
	router.HandleFunc("/api/parking-spots/{id}", ParkingSpotsGetById).Methods("GET")
	router.HandleFunc("/api/parking-spots/{id}", ParkingSpotsUpdate).Methods("PUT")
	router.HandleFunc("/api/parking-spots/{id}", ParkingSpotsDelete).Methods("DELETE")

	fmt.Println("start listening on PORT")
	err := http.ListenAndServe(":8081", router)
	if err != nil {
		fmt.Println("err came during listen")
	}
}
func main() {
	fmt.Println("running main")
	connectDB()
	registerRoutes()
}
