import 'dotenv/config';

const grpc = require('@grpc/grpc-js');
import { connect } from "mongoose";

import { backendProto } from 'common-utils';
import * as rpcImplementations from 'rpc';
import logger from 'utils/logging';

const PORT = process.env.CHAT_SERVICE_PORT || 31073;

async function start() {
    try {
        await connect(process.env.MONGODB_CONNECTION_STRING as string);
        let server = new grpc.Server();
        server.addService(backendProto.chat_service.ChatService.service, rpcImplementations);
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
