module github.com/bschirrmeister/Alexa-Radio/radio

go 1.15

replace github.com/bschirrmeister/Alexa-Radio/shared => ../shared

require (
	github.com/codegangsta/negroni v1.0.0
	github.com/go-sql-driver/mysql v1.6.0
	github.com/gorilla/mux v1.8.0
)

require (
	github.com/bschirrmeister/Alexa-Radio/shared v0.0.0-20230519192018-672f0ae3e6f5
	github.com/bschirrmeister/go-alexa v0.0.0-20230519192018-672f0ae3e6f5
)
