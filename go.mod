module github.com/jackwatson18/network-aprs-client

go 1.23.2

require (
	github.com/fatih/color v1.17.0
	github.com/mattn/go-sqlite3 v1.14.22
)

require (
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	golang.org/x/sys v0.19.0 // indirect
)

require internal/AX25 v1.0.0

replace internal/AX25 => ./internal/AX25
