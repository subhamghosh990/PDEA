package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

type Vehichle struct {
	ID            int       `json:"id"`
	SpotNumber    string    `json:"spot_number"`
	License_plate string    `json:"license_plate"`
	EntryTime     time.Time `json:"entry_time"`
	ExitTime      time.Time `json:"exit_time"`
}

type VehichleRes struct {
	ID            int    `json:"id"`
	SpotNumber    string `json:"spot_number"`
	License_plate string `json:"license_plate"`
	EntryTime     string `json:"entry_time,omitempty"`
	ExitTime      string `json:"exit_time,omitempty"`
}

type ParkingSpot struct {
	SpotNumber  string `json:"spot_number"`
	Type        string `json:"type"`
	IsAvailable bool   `json:"is_available"`
}

var (
	db *sql.DB
)

func connectDB() {
	// Define the connection string
	connStr := "host=localhost port=5432 user=robot password=cisco123 dbname=pdea sslmode=disable"
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
func converTime(t time.Time) string {
	return t.Format("02-01-2006 15:04:05")
}
func getAllSpotData() ([]ParkingSpot, error) {
	qr := `select spot_number, type, is_available from parking_rec`
	rows, err := db.Query(qr)
	if err != nil {
		return nil, err
	}
	var res []ParkingSpot
	for rows.Next() {
		var p ParkingSpot
		err := rows.Scan(&p.SpotNumber, &p.Type, &p.IsAvailable)
		if err != nil {
			continue
		}
		res = append(res, p)
	}
	return res, err
}
func updateParkingSpot(p ParkingSpot) {
	qr := `UPDATE parking_rec SET is_available = $1 where spot_number = $2;`
	db.Exec(qr, p.IsAvailable, p.SpotNumber)
}
func insertVechileEntry(v Vehichle) {
	qr := `INSERT INTO vehicle_records(id, spot_number, license_plate , entry_time) VALUES($1, $2, $3,$4);`
	db.Exec(qr)
}
func getAllVData() ([]Vehichle, error) {
	qr := `select id, spot_number, license_plate , entry_time, exit_time from vehicle_records;`
	rows, err := db.Query(qr)
	if err != nil {
		return nil, err
	}
	var res []Vehichle
	for rows.Next() {
		var v Vehichle
		rows.Scan(&v.ID, &v.SpotNumber, &v.License_plate, &v.EntryTime, &v.ExitTime)
		res = append(res, v)
	}
	return res, err
}
func updateExit(v Vehichle) {
	qr := `UPDATE vehicle_records SET exit_time = $1 where id = $2`
	db.Exec(qr, v.ExitTime, v.ID)
}
func RegisterEntry(w http.ResponseWriter, r *http.Request) {
	fmt.Println("RegisterEntry")
	var reqBody Vehichle
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		fmt.Println("entry invalid req")
		http.Error(w, "Invalid request Data", http.StatusBadRequest)
		return
	}
	spots, err := getAllSpotData()
	if err != nil {
		fmt.Println("server error entry")
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}
	found := false
	var Sp ParkingSpot
	for _, s := range spots {
		if s.SpotNumber == reqBody.SpotNumber {
			found = true
			Sp = s
		}
	}
	if !found {
		http.Error(w, "Parking spot not found", http.StatusNotFound)
		return
	}
	if !Sp.IsAvailable {
		http.Error(w, "Parking spot not found", http.StatusNotFound)
		return
	}

	vDatas, err := getAllVData()
	if err != nil {
		fmt.Println("server error entry")
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}
	var static_id int
	for _, d := range vDatas {
		if static_id <= d.ID {
			static_id = d.ID
		}
	}
	reqBody.EntryTime = time.Now()
	insertVechileEntry(reqBody)
	Sp.IsAvailable = false
	updateParkingSpot(Sp)
	res := VehichleRes{ID: reqBody.ID, SpotNumber: reqBody.SpotNumber, License_plate: reqBody.License_plate, EntryTime: converTime(reqBody.EntryTime)}
	resJson, _ := json.Marshal(res)
	w.WriteHeader(http.StatusCreated)
	w.Write(resJson)
}
func RegisterExit(w http.ResponseWriter, r *http.Request) {
	fmt.Println("RegisterExit")
	var reqBody Vehichle
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		fmt.Println("entry invalid req")
		http.Error(w, "Invalid request Data", http.StatusBadRequest)
		return
	}
	spots, err := getAllSpotData()
	if err != nil {
		fmt.Println("server error entry")
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}
	found := false
	var Sp ParkingSpot
	for _, s := range spots {
		if s.SpotNumber == reqBody.SpotNumber {
			found = true
			Sp = s
		}
	}
	if !found {
		http.Error(w, "Parking spot not found", http.StatusNotFound)
		return
	}

	vDatas, err := getAllVData()
	if err != nil {
		fmt.Println("server error entry")
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}
	found = false
	var v Vehichle
	for _, d := range vDatas {
		if reqBody.SpotNumber == d.SpotNumber && reqBody.License_plate == d.License_plate && d.ExitTime.IsZero() {
			found = true
			v = d
			break
		}
	}
	v.ExitTime = time.Now()
	updateExit(v)
	Sp.IsAvailable = true
	updateParkingSpot(Sp)
	res := VehichleRes{ID: v.ID, SpotNumber: v.SpotNumber, License_plate: v.License_plate, EntryTime: converTime(v.EntryTime), ExitTime: converTime(v.ExitTime)}
	resJson, _ := json.Marshal(res)
	w.WriteHeader(http.StatusOK)
	w.Write(resJson)
}
func GetVRecordsBySpotNo(w http.ResponseWriter, r *http.Request) {
	fmt.Println("GetVRecordsBySpotNo")
	id := r.URL.Path[len("/api/vehicle-records/"):]
	vDatas, err := getAllVData()
	if err != nil {
		fmt.Println("server error entry")
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}
	var res []VehichleRes
	for _, v := range vDatas {
		if v.SpotNumber == id {
			vr := VehichleRes{ID: v.ID, SpotNumber: v.SpotNumber, License_plate: v.License_plate, EntryTime: converTime(v.EntryTime)}
			if !v.ExitTime.IsZero() {
				vr.ExitTime = converTime(v.ExitTime)
			}
			res = append(res, vr)
		}
	}
	resJson, _ := json.Marshal(res)
	w.WriteHeader(http.StatusOK)
	w.Write(resJson)

}
func registerRoutes() {
	router := mux.NewRouter()
	router.HandleFunc("/api/vehicle-entries", RegisterEntry).Methods("POST")
	router.HandleFunc("/api/vehicle-exits", RegisterExit).Methods("POST")
	router.HandleFunc("/api/vehicle-records/{spot_no}", GetVRecordsBySpotNo).Methods("GET")
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
