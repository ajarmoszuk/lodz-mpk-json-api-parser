package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/antchfx/xmlquery"
	_ "github.com/mattn/go-sqlite3"
)

// Structure for cached timetable data
type Timetable struct {
	ID          int
	BusStopNo   int
	Data        []byte
	LastUpdated time.Time
}

func main() {
	db, err := sql.Open("sqlite3", "./cache.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Create table if it doesn't exist
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS timetable (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			bus_stop_no INTEGER,
			data BLOB,
			last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`)
	if err != nil {
		log.Fatal(err)
	}

	// HTTP handler
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Get bus stop number from URL parameter
		busStopNo, err := strconv.Atoi(r.URL.Query().Get("busStopNo"))
		if err != nil || busStopNo == 0 {
			response, _ := json.Marshal(map[string]string{"error": "Invalid bus stop number"})
			w.Write(response)
			return
		}

		// Delete cached records that are older than 1 minute
		_, err = db.Exec("DELETE FROM timetable WHERE last_updated < datetime('now', '-1 minutes')")
		if err != nil {
			log.Fatal(err)
		}

		// Check if cached data exists and is not expired
		var cachedData Timetable
		err = db.QueryRow("SELECT * FROM timetable WHERE bus_stop_no = ? AND last_updated > datetime('now', '-1 minutes')", busStopNo).Scan(
			&cachedData.ID,
			&cachedData.BusStopNo,
			&cachedData.Data,
			&cachedData.LastUpdated,
		)
		if err != nil && err != sql.ErrNoRows {
			log.Fatal(err)
		}

		// If cached data exists, return it
		if cachedData.ID > 0 {
			response := cachedData.Data
			w.Write(response)
			return
		}

		// Fetch new data from URL and cache it
		url := fmt.Sprintf("http://rozklady.lodz.pl/Home/GetTimetableReal?busStopNum=%d", busStopNo)

		// Create a new HTTP client with a custom header
		client := &http.Client{}
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			response, _ := json.Marshal(map[string]string{"error": "Invalid URL"})
			w.Write(response)
			log.Fatal(err)
			return
		}
		req.Header.Set("Accept", "application/xml")

		// Send the request and get the response
		resp, err := client.Do(req)
		if err != nil {
			response, _ := json.Marshal(map[string]string{"error": "No data found for the given URL"})
			w.Write(response)
			log.Fatal(err)
			return
		}
		defer resp.Body.Close()

		// Load the response body into an XML document
		doc, err := xmlquery.Parse(resp.Body)
		if err != nil {
			response, _ := json.Marshal(map[string]string{"error": "Invalid XML document"})
			w.Write(response)
			log.Fatal(err)
			return
		}
		
		// Find the timetable data in the XML document
		schedules := xmlquery.FindOne(doc, "//Stop/Day")
		if schedules == nil {
			response, _ := json.Marshal(map[string]string{"error": "No data found for the given bus stop number"})
			w.Write(response)
			return
		}

		// Extract timetable data from the XML document
		var output []map[string]string
		for _, schedule := range xmlquery.Find(schedules, "//R") {
			jsonOutput := map[string]string{}
			jsonOutput["route_number"] = xmlquery.FindOne(schedule, "//@nr").InnerText()
			jsonOutput["route_direction"] = xmlquery.FindOne(schedule, "//@dir").InnerText()
			vehicleType := xmlquery.FindOne(schedule, "//@vt").InnerText()
			if vehicleType == "A" {
				jsonOutput["vehicle_type"] = "BUS"
			} else if vehicleType == "T" {
				jsonOutput["vehicle_type"] = "TRAM"
			} else {
				jsonOutput["vehicle_type"] = vehicleType
			}
			estimatedTimeSeconds, err := strconv.ParseInt(xmlquery.FindOne(schedule, "//S/@s").InnerText(), 10, 64)
			if err != nil {
				jsonOutput["estimated_time"] = "Unknown"
				jsonOutput["human_estimated_time"] = "Unknown"
			} else {
				estimatedTime := time.Now().Add(time.Duration(estimatedTimeSeconds) * time.Second)
				jsonOutput["estimated_time"] = estimatedTime.Format(time.RFC3339)
				timeDiff := estimatedTime.Sub(time.Now())
				if timeDiff < 0 {
					jsonOutput["human_estimated_time"] = "Now"
				} else if timeDiff < time.Minute {
					if timeDiff.Seconds() == 1 {
						jsonOutput["human_estimated_time"] = "in 1 second"
					} else {
						jsonOutput["human_estimated_time"] = fmt.Sprintf("in %d seconds", int(timeDiff.Seconds()))
					}
				} else if timeDiff < time.Hour {
					if timeDiff.Minutes() == 1 {
						jsonOutput["human_estimated_time"] = "in 1 minute"
					} else {
						jsonOutput["human_estimated_time"] = fmt.Sprintf("in %d minutes", int(timeDiff.Minutes()))
					}
				} else {
					if timeDiff.Hours() == 1 {
						jsonOutput["human_estimated_time"] = "in 1 hour"
					} else {
						jsonOutput["human_estimated_time"] = fmt.Sprintf("in %d hours", int(timeDiff.Hours()))
					}
				}				
			}
			output = append(output, jsonOutput)
		}

		// Convert the output to JSON
		jsonMap := map[string][]map[string]string{"timetable": output}
		jsonData, err := json.Marshal(jsonMap)
		if err != nil {
			log.Fatal(err)
		}

		// Insert new data into the database
		_, err = db.Exec("INSERT INTO timetable(bus_stop_no, data) VALUES(?, ?)", busStopNo, jsonData)
		if err != nil {
			log.Fatal(err)
		}

		// Return the timetable data
		response, err := json.Marshal(jsonMap)
		if err != nil {
			log.Fatal(err)
		}

		w.Write(response)
	})

	// Start the HTTP server
	http.ListenAndServe(":8080", nil)
}
