module secure-ops-pipeline/services/device-gateway

go 1.22.2

replace golang.org/x/sync => github.com/golang/sync v0.7.0

replace golang.org/x/text => github.com/golang/text v0.15.0

require github.com/go-zeromq/zmq4 v0.17.0

require (
	github.com/go-zeromq/goczmq/v4 v4.2.2 // indirect
	golang.org/x/sync v0.7.0 // indirect
	golang.org/x/text v0.15.0 // indirect
)
