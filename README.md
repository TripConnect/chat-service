# Introduction
The backend micro-service for chatting related

# Starting up service
```sh
go mod download # Install packages
go run . # Run service server
```

# Cheatsheet
Verify build availability
```sh
docker build -t chat-service:latest . # Build image
docker run -d -p 31074:31074 --name chat-service chat-service:latest # Run container
```
