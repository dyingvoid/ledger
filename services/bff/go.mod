module github.com/dyingvoid/ledger/services/bff

go 1.27.1

require (
	github.com/dyingvoid/ledger/shared v0.0.0
	github.com/joho/godotenv v1.5.1
	github.com/kelseyhightower/envconfig v1.4.0
	google.golang.org/grpc v1.85.0
)

require (
	golang.org/x/net v0.60.0 // indirect
	golang.org/x/sys v0.49.0 // indirect
	golang.org/x/text v0.43.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260817212433-ac3dfec99bb1 // indirect
	google.golang.org/protobuf v1.36.12 // indirect
)

replace github.com/dyingvoid/ledger/shared => ../../shared
