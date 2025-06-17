# Introduction
The backend micro-service for chatting related

# Starting up service
```sh
go run .
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
## CLI
**Starting up infa**
```sh
docker-compose up # up
docker-compose down # down
```
**Using cassandra CQL**
```sh
docker exec -it <container-id> sh
cqlsh
use ks_chat;
```
