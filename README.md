# Introduction
The backend micro-service for chatting related

# Starting up service
```sh
go run ./src
```

# Build
## gRPC
Build gRPC related
```sh
protoc --go_out=src/protos/defs --go_opt=paths=source_relative --go-grpc_out=src/protos/defs --go-grpc_opt=paths=source_relative --proto_path=src/protos src/protos/chat_service.proto
```
Target tree will like here
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

# Development cheatsheet
## Installation
**Commands**  
`go install`: Install globally  
`go get`: Install locally  
**Install gRPC core packages**
```sh
go get google.golang.org/grpc
```
**Setup gRPC protoc generator**
```sh
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
export PATH=$PATH:$HOME/go/bin
```
**Starting up infa**
```sh
docker-compose up # up
docker-compose down # down
```
