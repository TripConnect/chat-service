const grpc = require('@grpc/grpc-js');
import { Timestamp } from "google-protobuf/google/protobuf/timestamp_pb";
import { v4 as uuidv4 } from 'uuid';

import Conversations, { IConversation } from "./database/models/conversations";
import Messages from './database/models/messages';
import { Conversation, ConversationType } from "./type";
import logger from "./utils/logging";

export async function createConversation(call: any, callback: any) {
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

export async function createChatMessage(call: any, callback: any) {
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

export async function searchConversations(call: any, callback: any) {
    try {
        let { type, memberIds, term, page_number, page_limit, message_page_limit } = call.request;
        let result: any = [];
        let conversations: IConversation[] = [];

        conversations = await Conversations
            .find({ members: { $in: memberIds } })
            .sort({ createdAt: 1 })
            .skip(page_limit * (Math.abs(page_number) - 1))
            .limit(page_limit)
            .exec();

        for (let conversation of conversations) {
            let messages = await Messages
                .find({ conversationId: conversation.conversationId })
                .sort({ createdAt: -1 })
                .limit(message_page_limit)
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

export async function findConversation(call: any, callback: any) {
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
