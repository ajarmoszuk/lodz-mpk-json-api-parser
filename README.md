# Timetable Cache and Retrieval for Łódź MPK XML API
[![Go](https://github.com/ajarmoszuk/lodz-mpk-xml-parser-api/actions/workflows/go.yml/badge.svg)](https://github.com/ajarmoszuk/lodz-mpk-xml-parser-api/actions/workflows/go.yml)

This Go application provides a simple HTTP server that caches and retrieves timetable data for bus and/or tram stops. It uses an SQLite database to store cached data and fetches new data from an external API when needed and returns the data in a JSON format. 

Below is an overview of the code and its functionality.
# Prerequisites

Before running the application, make sure you have the following prerequisites installed:

- Go (Golang)
- SQLite3

# Installation
Clone the repository:

    git clone https://github.com/ajarmoszuk/lodz-mpk-xml-parser-api.git
    cd lodz-mpk-xml-parser-api

Build the Go application:

    go get
    go build

Run the application:

    ./lodz-mpk-xml-parser-api

By default, the application listens on port 8080. You can change the port in the http.ListenAndServe(":8080", nil) line at the end of the code.

# Usage
### Cache Retrieval

You can retrieve cached timetable data by making an HTTP GET request to the root ("/") endpoint with the busStopNo query parameter. For example:

    GET /?busStopNo=573

Replace 573 with the desired bus or tram stop number. You can retrieve the ID of the desired bus or tram stop by going to http://rozklady.lodz.pl/ and clicking on the desired number on the map, then a new window will appear with the ID that you can copy over. If the data is available in the cache and not expired, it will be returned in JSON format. If not, the application will fetch new data from the external API, cache it, and then return it.
# Error Handling

The application handles various error scenarios and returns JSON error responses with appropriate status codes.
# Structure

    main.go: The main application code, including the HTTP server setup, caching logic, and error handling.
    cache.db: SQLite database file for storing cached timetable data.

# Dependencies

    github.com/antchfx/xmlquery: Used to parse XML data from the external API.
    github.com/mattn/go-sqlite3: SQLite driver for Go.

# Author

This code was written by Alex Jarmoszuk.
# License

This project is licensed under the GPL-3.0 License. See the LICENSE file for details.
