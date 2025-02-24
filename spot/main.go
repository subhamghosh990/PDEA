// podman run -dit --restart on-failure --log-driver=json-file --log-opt max-file=2  --log-opt max-size=20m --name robot-postgres -p 5432:5432 -e POSTGRES_DB=pdea -e POSTGRES_USER=robot -e POSTGRES_PASSWORD=cisco123 postgres:11.5
package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

type ParkingSpot struct {
	ID          int    `json:"id"`
	SpotNumber  string `json:"spot_number"`
	Type        string `json:"type"`
	IsAvailable string `json:"is_available"`
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
CREATE TABLE IF NOT EXISTS parking_spots (
id SERIAL PRIMARY KEY,
spot_number TEXT ,
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

func getParkinspotsDataAll() ([]ParkingSpot, error) {
	var res []ParkingSpot
	qr := `select id, spot_number, type, is_available from parking_spots`
	rows, err := db.Query(qr)
	if err != nil {
		return res, err
	}
	for rows.Next() {
		var p ParkingSpot
		err := rows.Scan(&p.ID, &p.SpotNumber, &p.Type, &p.IsAvailable)
		if err != nil {
			continue
		}
		res = append(res, p)
	}
	return res, err
}
func insertParkData(p ParkingSpot) {
	qr := `INSERT INTO parking_spots(id, spot_number, type, is_available) VALUES ($1, $2, $3, $4)`
	db.Exec(qr, p.ID, p.SpotNumber, p.Type, p.IsAvailable)
}
func ParkingSpotsEntry(w http.ResponseWriter, r *http.Request) {
	var reqBody ParkingSpot
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		http.Error(w, "Invalid request bodt", http.StatusBadRequest)
		return
	}
	datas, err := getParkinspotsDataAll()
	if err != nil {
		fmt.Println("get query error on entry")
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	var static int
	for _, d := range datas {
		if d.SpotNumber == reqBody.SpotNumber {
			fmt.Println("Duplicate entry")
			http.Error(w, "Spot is already exist", http.StatusConflict)
			return
		}
		if static <= d.ID {
			static = d.ID
		}
	}
	insertParkData(reqBody)
	w.WriteHeader(http.StatusCreated)

}
func ParkingSpotsGetAll(w http.ResponseWriter, r *http.Request) {
	fmt.Println("ParkingSpotsGetAll")
	datas, err := getParkinspotsDataAll()
	if err != nil {
		fmt.Println("get all parking data error ", err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	resJason, _ := json.Marshal(datas)
	w.WriteHeader(http.StatusOK)
	w.Write(resJason)
}

func ParkingSpotsGetById(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/api/parking-spots/"):]
	idInt, _ := strconv.Atoi(id)
	datas, err := getParkinspotsDataAll()
	if err != nil {
		fmt.Println("get all parking data error ", err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	for _, d := range datas {
		if d.ID == idInt {
			resJson, _ := json.Marshal(d)
			w.WriteHeader(http.StatusOK)
			w.Write(resJson)
			return
		}
	}
	http.Error(w, "Id not found", http.StatusNotFound)
}

func updateParkingData(p ParkingSpot) {
	qr := `UPDATE parking_spots SET type = $1 and is_available = $2 where id = $3;`
	db.Exec(qr, p.Type, p.IsAvailable, p.ID)
}
func ParkingSpotsUpdate(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/api/parking-spots/"):]
	idInt, _ := strconv.Atoi(id)
	datas, err := getParkinspotsDataAll()
	if err != nil {
		fmt.Println("update parking data error ", err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	var reqBody ParkingSpot
	if err = json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		fmt.Println("update parking data decode error ", err)
		http.Error(w, "server error", http.StatusBadRequest)
		return
	}
	found := false
	var p ParkingSpot
	for _, d := range datas {
		if d.ID == idInt {
			found = true
			p = d
		}
	}
	if !found {
		http.Error(w, "Parking sopt not found", http.StatusNotFound)
		return
	}
	p.IsAvailable = reqBody.IsAvailable
	p.Type = reqBody.Type
	updateParkingData(p)
	resJson, _ := json.Marshal(p)
	w.WriteHeader(http.StatusAccepted)
	w.Write(resJson)
}
func deleteParkingData(p ParkingSpot) {
	qr := `DELETE from parking_spots where id = $1;`
	db.Exec(qr, p.ID)
}
func ParkingSpotsDelete(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/api/parking-spots/"):]
	idInt, _ := strconv.Atoi(id)
	datas, err := getParkinspotsDataAll()
	if err != nil {
		fmt.Println("update parking data error ", err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	found := false
	var p ParkingSpot
	for _, d := range datas {
		if d.ID == idInt {
			found = true
			p = d
		}
	}
	if !found {
		http.Error(w, "Parking sopt not found", http.StatusNotFound)
		return
	}
	deleteParkingData(p)
	w.WriteHeader(http.StatusOK)
	res := "Parking spot has been deleted successfully."
	resJson, _ := json.Marshal(res)
	w.Write(resJson)
}

func registerRoutes() {
	router := mux.NewRouter()

	router.HandleFunc("/api/parking-spots", ParkingSpotsEntry).Methods("POST")
	router.HandleFunc("/api/parking-spots/all", ParkingSpotsGetAll).Methods("GET")
	router.HandleFunc("/api/parking-spots/{id}", ParkingSpotsGetById).Methods("GET")
	router.HandleFunc("/api/parking-spots/{id}", ParkingSpotsUpdate).Methods("PUT")
	router.HandleFunc("/api/parking-spots/{id}", ParkingSpotsDelete).Methods("DELETE")

	fmt.Println("start listening on PORT")
	err := http.ListenAndServe(":8080", router)
	if err != nil {
		fmt.Println("err came during listen")
	}
}
func main() {
	fmt.Println("running main")
	connectDB()
	registerRoutes()
}
