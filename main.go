package main

import (
	"database/sql"
	"log"
	"os"
	"time"

	"github.com/jackwatson18/network-aprs-client/Aprs"
	_ "github.com/mattn/go-sqlite3"
)

func createDB() {
	db, err := sql.Open("sqlite3", "./aprs.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	sqlStatement := `
	CREATE TABLE IF NOT EXISTS aprs (
		id integer not null primary key,
		send_callsign TEXT,
		dest_callsign TEXT,
		longitude REAL,
		latitude REAL,
		heading INTEGER,
		speed INTEGER,
		altitude INTEGER,
		comment TEXT,
		symbolTableId TEXT,
		symbolId TEXT,
		raw TEXT
	)`

	_, err = db.Exec(sqlStatement)
	if err != nil {
		log.Printf("%q: %s\n", err, sqlStatement)
	}
}

func main() {
	server := "localhost:8001"
	if len(os.Args) > 1 {
		server = os.Args[1]
	}
	go Aprs.ConnectionLoop(server)

	createDB()

	for {
		time.Sleep(time.Second)
	}

}
