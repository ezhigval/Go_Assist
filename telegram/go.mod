module telegram

go 1.21

require (
	databases v0.0.0
	github.com/go-telegram-bot-api/telegram-bot-api/v5 v5.5.1
	modulr v0.0.0
)

require (
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/pgx/v5 v5.7.1 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	golang.org/x/crypto v0.27.0 // indirect
	golang.org/x/sync v0.8.0 // indirect
	golang.org/x/text v0.18.0 // indirect
)

replace databases => ../databases

replace github.com/davecgh/go-spew v1.1.0 => github.com/davecgh/go-spew v1.1.1

replace modulr => ..
