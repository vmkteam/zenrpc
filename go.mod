module github.com/vmkteam/zenrpc/v2

go 1.18

require (
	github.com/gorilla/websocket v1.4.2
	github.com/prometheus/client_golang v1.13.0
	github.com/smartystreets/goconvey v1.6.4
	golang.org/x/text v0.10.0
	golang.org/x/tools v0.10.0
)

require (
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.1.2 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/gopherjs/gopherjs v0.0.0-20181017120253-0766667cb4d1 // indirect
	github.com/jtolds/gls v4.20.0+incompatible // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.1 // indirect
	github.com/prometheus/client_model v0.2.0 // indirect
	github.com/prometheus/common v0.37.0 // indirect
	github.com/prometheus/procfs v0.8.0 // indirect
	github.com/smartystreets/assertions v0.0.0-20180927180507-b2de0cb4f26d // indirect
	golang.org/x/mod v0.11.0 // indirect
	golang.org/x/sys v0.9.0 // indirect
	google.golang.org/protobuf v1.28.1 // indirect
)

retract (
	v2.2.10 //invalid version cached
	v2.2.5
)
