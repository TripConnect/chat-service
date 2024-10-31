import 'dotenv/config';
const fs = require('fs');
import { v4 as uuidv4 } from 'uuid';
import logger from './utils/logging';
const grpc = require('@grpc/grpc-js');
const protoLoader = require('@grpc/proto-loader');
import { connect } from "mongoose";
import Conversations, { ConversationType, IConversation } from './mongo/models/conversations';
import Messages from './mongo/models/messages';
import { Timestamp } from 'google-protobuf/google/protobuf/timestamp_pb';

let packageDefinition = protoLoader.loadSync(
    require.resolve('common-utils/protos/backend.proto'),
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
    type: ConversationType;
    name: string;
    memberIds: string[];
    messages: ChatMessage[];
    createdAt: Date;
}

const PORT = process.env.USER_SERVICE_PORT || 31073;

async function createConversation(call: any, callback: any) {
    let { ownerId, name, type, memberIds } = call.request;

    let conversation: IConversation | null = null;

    if (memberIds.length === 0) {
        callback({
            code: grpc.status.INVALID_ARGUMENT,
            message: 'Conversation members cannot be empty'
        });
        return;
    }

    try {
        switch (type) {
            case ConversationType.PRIVATE:
                name = null;
                ownerId = null;

                let existConversation = await Conversations
                    .findOne({ type: ConversationType.PRIVATE, members: { $all: memberIds } })
                    .exec();

                if (existConversation) conversation = existConversation;
                break;
        }

        if (conversation == null) {
            conversation = await Conversations.create({
                conversationId: uuidv4(),
                name,
                type,
                members: memberIds,
                createdBy: ownerId,
                lastMessageAt: null,
            });
        }

        let conversationResponse: Conversation = {
            id: conversation.conversationId as string,
            type: conversation.type,
            name: conversation.name as string,
            memberIds: conversation.members as string[],
            messages: [],
            createdAt: conversation.createdAt,
        }
        callback(null, conversationResponse);
    } catch (error: any) {
        logger.error(error.message);
        callback(error, null);
    }
}

async function createChatMessage(call: any, callback: any) {
    try {
        let { conversationId, fromUserId, messageContent } = call.request;

        let conversation = await Conversations.findOne({ conversationId });
        if (!conversation) {
            callback({
                code: grpc.status.NOT_FOUND,
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
            createdAt: Timestamp.fromDate(message.createdAt),
        };
        callback(null, messageResponse);
    } catch (error: any) {
        logger.error(error.message);
        callback(error, null);
    }
}

async function searchConversations(call: any, callback: any) {
    try {
        let { type, memberIds, term, page, limit, messageLimit } = call.request;
        let result: any = [];
        let conversations: IConversation[] = [];

        conversations = await Conversations
            .find({ members: { $in: memberIds } })
            .sort({ createdAt: 1 })
            .skip(limit * (Math.abs(page) - 1))
            .limit(limit)
            .exec();

        for (let conversation of conversations) {
            let messages = await Messages
                .find({ conversationId: conversation.conversationId })
                .sort({ createdAt: -1 })
                .limit(messageLimit)
                .exec();

            result.push({
                id: conversation.conversationId,
                name: conversation?.name,
                type: conversation?.type,
                createdBy: null,
                createdAt: conversation?.createdAt,
                lastMessageAt: null,
                memberIds: conversation.members,
                messages: messages.map(({ messageId, fromUserId, messageContent, createdAt }) => {
                    return {
                        id: messageId,
                        conversationId: conversation.conversationId,
                        fromUserId,
                        messageContent,
                        createdAt,
                    }
                }),
            });
        }
        callback(null, { conversations: result });
    } catch (error: any) {
        logger.error(error.message);
        callback(error, null);
    }
}

async function findConversation(call: any, callback: any) {
    try {
        let { conversationId, messagePage = 1, messageLimit = 1 } = call.request;
        let conversation = await Conversations.findOne({ conversationId });

        if (!conversation) {
            callback({
                code: grpc.status.NOT_FOUND,
                message: 'Conversation not found'
            });
            return;
        }

        let messages = await Messages
            .find({ conversationId })
            .sort({ createdAt: -1 })
            .skip(messageLimit * (Math.abs(messagePage) - 1))
            .limit(messageLimit)
            .exec();

        let conversationResponse = {
            id: conversation.conversationId,
            name: conversation.name,
            type: conversation.type,
            createdBy: conversation.createdBy,
            createdAt: Timestamp.fromDate(conversation.createdAt).toObject(),
            lastMessageAt: conversation.lastMessageAt && Timestamp.fromDate(conversation.lastMessageAt),
            memberIds: conversation.members,
            messages: messages.map(({ messageId, fromUserId, messageContent, createdAt }) => {
                return {
                    id: messageId,
                    conversationId: conversation.conversationId,
                    fromUserId,
                    messageContent,
                    createdAt: Timestamp.fromDate(createdAt).toObject(),
                }
            }),
        };
        callback(null, conversationResponse);
    } catch (error: any) {
        logger.error(error.message);
        callback(error, null);
    }
}

async function start() {
    try {
        await connect(process.env.MONGODB_CONNECTION_STRING as string);
        let server = new grpc.Server();
        server.addService(backendProto.Chat.service, {
            createConversation,
            createChatMessage,
            searchConversations,
            findConversation,
        });
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
