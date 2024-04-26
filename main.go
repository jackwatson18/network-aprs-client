package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackwatson18/network-aprs-client/Aprs"
	_ "github.com/mattn/go-sqlite3"
)

type PacketsPOSTbody struct {
	Callsign  string
	Ssid      int
	Latitude  float64
	Longitude float64
	Comment   string
}

var server string = "localhost:8001"

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
		send_ssid INTEGER,
		dest_callsign TEXT,
		dest_ssid INTEGER,
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

func dbGetPackets() []Aprs.APRS_Packet {
	db, err := sql.Open("sqlite3", "./aprs.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	rows, err := db.Query("SELECT * FROM aprs ORDER BY id DESC")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	packets := make([]Aprs.APRS_Packet, 0)
	for rows.Next() {
		packet := Aprs.APRS_Packet{}
		err = rows.Scan(
			&packet.Id,
			&packet.Original_AX25.Src.Callsign,
			&packet.Original_AX25.Src.Ssid,
			&packet.Original_AX25.Dst.Callsign,
			&packet.Original_AX25.Dst.Ssid,
			&packet.Longitude,
			&packet.Latitude,
			&packet.Heading,
			&packet.Speed,
			&packet.Altitude,
			&packet.Comment,
			&packet.SymbolTableId,
			&packet.SymbolId,
			&packet.Raw,
		)

		if err != nil {
			log.Fatal(err)
		}
		packets = append(packets, packet)
	}

	return packets

}

func getPackets(c *gin.Context) {
	packets := dbGetPackets()
	c.IndentedJSON(http.StatusOK, packets)
}

func recievePacket(c *gin.Context) {
	data := PacketsPOSTbody{}
	err := c.BindJSON(&data)
	if err != nil {
		fmt.Println(err)
		c.IndentedJSON(http.StatusBadRequest, "Could not transmit packet")
		return
	} else {
		fmt.Println(data)
	}

	newAPRS := Aprs.APRS_Packet{
		Src_callsign:  data.Callsign,
		Src_ssid:      uint8(data.Ssid),
		Latitude:      data.Latitude,
		Longitude:     data.Longitude,
		Comment:       data.Comment,
		SymbolTableId: "/",
		SymbolId:      "-",
	}

	axframe, err := Aprs.APRS_to_AX25(newAPRS)
	if err != nil {
		fmt.Println(err)
		c.IndentedJSON(http.StatusBadRequest, "Could not transmit packet")
		return
	}
	rawbytes := Aprs.ModifiedAX25_to_bytes(axframe)
	Aprs.SendToModem(server, rawbytes)

	c.IndentedJSON(http.StatusCreated, "Packet accepted")
}

func router() {
	router := gin.Default()

	router.StaticFS("/static", http.Dir("./static"))
	// router.Use(static.Serve("/", static.LocalFile("static", true)))

	// router.GET("/", func(c *gin.Context) {
	// 	c.HTML(http.StatusOK, "index.html", gin.H{})
	// })

	router.GET("/packets", getPackets)
	router.POST("/packets", recievePacket)

	router.Run("localhost:8080")
}

func main() {
	if len(os.Args) > 1 {
		server = os.Args[1]
	}

	// Aprs.TestEncodeAndSend(server)
	// Aprs.TestCallsignShifting()
	go Aprs.ConnectionLoop(server)
	createDB()

	go router()

	for {
		time.Sleep(time.Second)
	}

}
