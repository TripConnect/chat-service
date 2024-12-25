import 'dotenv/config';

import { connect } from "mongoose";
import { v4 as uuidv4 } from 'uuid';

import Messages from "../models/messages";
import Conversations from "../models/conversations";
import { ConversationType } from "../../type";

(async function () {
    const firstUserId = "00000000-0000-0000-0000-000000000001";
    const secondUserId = "00000000-0000-0000-0000-000000000002";
    const numberOfMessages = 200;

    await connect(process.env.MONGODB_CONNECTION_STRING as string);

    let conversation = await Conversations.create({
        conversationId: uuidv4(),
        name: null,
        type: ConversationType.PRIVATE,
        members: [firstUserId, secondUserId],
        createdBy: firstUserId,
        lastMessageAt: null,
    });

    for (let i = 1; i <= numberOfMessages; i++) {
        let senderId = i % 2 === 0 ? firstUserId : secondUserId;
        let fakeMessageContent = `Hi ${i}`;

        await Messages.create({
            conversationId: conversation.conversationId,
            fromUserId: senderId,
            messageContent: fakeMessageContent,
        });
    }
})();
