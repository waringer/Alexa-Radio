module github.com/waringer/Alexa-Radio/scanner

go 1.17

replace github.com/waringer/Alexa-Radio/shared => ../shared

require (
	github.com/dhowden/tag v0.0.0-20201120070457-d52dcb253c63
	github.com/go-sql-driver/mysql v1.6.0
	github.com/vmware/go-nfs-client v0.0.0-20190605212624-d43b92724c1b
	github.com/waringer/Alexa-Radio/shared v0.0.0-00010101000000-000000000000
)

require github.com/rasky/go-xdr v0.0.0-20170124162913-1a41d1a06c93 // indirect
