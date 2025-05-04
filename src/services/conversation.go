package services

import pb "github.com/TripConnect/chat-service/src/protos/defs"

func CreateConversation(req *pb.CreateConversationRequest) *pb.Conversation {
	result := &pb.Conversation{
		Name: req.GetName(),
	}
	return result
}

func SearchConversations(req *pb.SearchConversationsRequest) *pb.Conversations {
	result := &pb.Conversations{}
	return result
}
