module neuroboost/api-go

go 1.22

require (
	github.com/go-chi/chi/v5 v5.0.12
	github.com/go-chi/cors v1.2.1
	github.com/golang-jwt/jwt/v5 v5.2.1
	github.com/jackc/pgx/v5 v5.5.5
	github.com/zemdenalex/neuroboost-bot v0.0.0
	golang.org/x/crypto v0.21.0
)

require (
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20221227161230-091c0ba34f0a // indirect
	github.com/jackc/puddle/v2 v2.2.1 // indirect
	golang.org/x/sync v0.1.0 // indirect
	golang.org/x/text v0.14.0 // indirect
)

// The line parser lives in the bot module (bot/parse) and is shared, not
// copied: a typed line must mean the same thing in the bot and on the web.
// ⚠ This is why the api image builds from the repo root (docker-compose*.yml).
replace github.com/zemdenalex/neuroboost-bot => ../bot
