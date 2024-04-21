import 'dotenv/config';
const fs = require('fs');
import { v4 as uuidv4 } from 'uuid';
import logger from './utils/logging';
const grpc = require('@grpc/grpc-js');
const protoLoader = require('@grpc/proto-loader');
import { connect } from "mongoose";
import Conversations, { ConversationType } from './mongo/models/conversations';
import Messages from './mongo/models/messages';

let packageDefinition = protoLoader.loadSync(
    process.env.PROTO_URL,
    {
        keepCase: true,
        longs: String,
        enums: String,
        defaults: true,
        oneofs: true
    });
let backendProto = grpc.loadPackageDefinition(packageDefinition).backend;

type ChatMessage = {
    id: string,
    conversationId: string,
    fromUserId: string,
    messageContent: string,
    createdAt: Date
}

type Conversation = {
    id: string;
    name: string;
    memberIds: string[];
    messages: ChatMessage[];
}

const PORT = process.env.USER_SERVICE_PORT || 31073;

async function createConversation(call: any, callback: any) {
    let { ownerId, name, type, memberIds } = call.request;

    if (type === ConversationType.PRIVATE) {
        name = null;
        ownerId = null;
    }

    try {
        let conversation = await new Conversations({
            conversationId: uuidv4(),
            name,
            type,
            members: memberIds,
            createdBy: ownerId,
            lastMessageAt: null,
        }).save();

        let conversationResponse: Conversation = {
            id: conversation.conversationId as string,
            name: conversation.name as string,
            memberIds: conversation.members as string[],
            messages: [],
        }
        callback(null, conversationResponse);
    } catch (error) {
        logger.error(error);
        callback(error, null);
    }
}

async function createChatMessage(call: any, callback: any) {
    try {
        let { conversationId, fromUserId, messageContent } = call.request;

        let conversation = await Conversations.findOne({ conversationId });
        if (!conversation) {
            callback({
                code: grpc.status.INVALID_ARGUMENT,
                message: 'Conversation not found'
            });
            return;
        }

        let message = await Messages.create({
            conversationId,
            fromUserId,
            messageContent,
        });
        let messageResponse = {
            id: message.id,
            conversationId: message.conversationId,
            fromUserId: message.fromUserId,
            messageContent: message.messageContent,
            createdAt: message.createdAt,
        };
        callback(null, messageResponse);
    } catch (error) {
        logger.error(`gRPC server stopped: ${error}`);
        callback(error, null);
    }

}

async function start() {
    try {
        await connect(process.env.MONGODB_CONNECTION_STRING as string);
        let server = new grpc.Server();
        server.addService(backendProto.Chat.service, { CreateConversation: createConversation, CreateChatMessage: createChatMessage });
        server.bindAsync(`0.0.0.0:${PORT}`, grpc.ServerCredentials.createInsecure(), (err: any, port: any) => {
            if (err != null) {
                return console.error(err);
            }
            console.log(`gRPC listening on ${PORT}`)
        });
    } catch (err) {
        logger.error(`gRPC server stopped: ${err}`);
    }
}

start();
