export enum ConversationType {
    PRIVATE = 'PRIVATE',
    GROUP = 'GROUP',
}

export type ChatMessage = {
    id: string,
    conversationId: string,
    fromUserId: string,
    messageContent: string,
    createdAt: Date
}

export type Conversation = {
    id: string;
    type: ConversationType;
    name: string;
    memberIds: string[];
    messages: ChatMessage[];
    createdAt: Date;
}
