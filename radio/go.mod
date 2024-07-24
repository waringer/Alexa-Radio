module github.com/waringer/Alexa-Radio/radio

go 1.17

replace github.com/waringer/Alexa-Radio/shared => ../shared

require (
	github.com/codegangsta/negroni v1.0.0
	github.com/go-sql-driver/mysql v1.6.0
	github.com/gorilla/mux v1.8.0
	github.com/waringer/Alexa-Radio/shared v0.0.0-00010101000000-000000000000
	github.com/waringer/go-alexa v0.0.0-20240724022700-ebf9a5e04ef2
)

require (
	github.com/rasky/go-xdr v0.0.0-20170124162913-1a41d1a06c93 // indirect
	github.com/vmware/go-nfs-client v0.0.0-20190605212624-d43b92724c1b // indirect
)
