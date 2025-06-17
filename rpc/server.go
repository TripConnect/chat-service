package rpc

import (
	"github.com/tripconnect/go-proto-lib/protos"
)

type Server struct {
	protos.UnimplementedChatServiceServer
}
