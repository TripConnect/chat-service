import mongoose, { Document, Schema, Model } from 'mongoose';
import { v4 as uuidv4 } from 'uuid';

// Define a TypeScript interface for the User document
interface IMessages extends Document {
    messageId: String,
    conversationId: String,
    fromUserId: String,
    messageContent: String,
    createdAt: Date,
}

const MessagesSchema = new Schema<IMessages>({
    messageId: {
        type: String,
        default: uuidv4,
        unique: true,
    },
    conversationId: String,
    fromUserId: String,
    messageContent: String,
    createdAt: {
        type: Date,
        default: Date.now,
    },
});

const Messages: Model<IMessages> = mongoose.model('messages', MessagesSchema);

export default Messages;
