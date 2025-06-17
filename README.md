# Introduction
The backend micro-service for chatting related

# Starting up service
```sh
go run .
```

# Development cheatsheet
## Installation
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
