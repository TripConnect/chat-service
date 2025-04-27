# Build proto
Simple gen
```sh
protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative chat_service.proto
```
Pretty dir change (recommended)
```sh
protoc --go_out=src/protos/defs --go_opt=paths=source_relative --go-grpc_out=src/protos/defs --go-grpc_opt=paths=source_relative --proto_path=src/protos src/protos/chat_service.proto
```
## Target
```sh
.
├── go.mod
├── go.sum
├── README.md
└── src
    ├── application.go
    └── protos
        ├── chat_service.proto
        └── defs
            ├── chat_service_grpc.pb.go
            └── chat_service.pb.go

4 directories, 7 files
```

# Installation
`go install`: Install globally
`go get`: Install locally
## Install core packages
```sh
go get google.golang.org/grpc
```
```sh
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
export PATH=$PATH:$HOME/go/bin
```

if search by members -> use [combine]: (userIdA)(userIdB)

# Start
```sh
go run .
```

# Quick command
```sh
rm -rf *
```