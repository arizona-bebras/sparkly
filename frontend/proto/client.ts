import { createClient } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import { PUBLIC_API_BASE_URL } from "$env/static/public";
import { ElizaService } from "./src/gen/connectrpc/eliza/v1/eliza_pb";

const transport = createConnectTransport({
	baseUrl: PUBLIC_API_BASE_URL || "https://demo.connectrpc.com",
});

export const client = createClient(ElizaService, transport);
