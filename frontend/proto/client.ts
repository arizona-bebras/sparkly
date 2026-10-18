import {
	Code,
	ConnectError,
	createClient,
	createRouterTransport,
} from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import { PUBLIC_API_BASE_URL } from "$env/static/public";
import { UserService } from "./gen/backend/proto/service_pb";

const mockTransport = createRouterTransport(({ service }) => {
	service(UserService, {
		async checkAuth(req) {
			console.log("⚡ [MOCK] checkAuth вызван");
			return { auth: true };
		},
		async getToken(req) {
			console.log("⚡ [MOCK] getToken для:", req.email);
			console.log(req.email, req.password);
			await new Promise((r) => setTimeout(r, 1600));
			if (req.email !== "admin@mail.ru" && req.password !== "admin") {
				throw new ConnectError(
					"Неверный логин или пароль!",
					Code.PermissionDenied,
				);
			}

			return {
				email: req.email,
				token: "mock-jwt-token-xyz",
				refreshToken: "mock-refresh-token-xyz",
				expiresIn: 3600n,
			};
		},
		async reg(req) {
			console.log("⚡ [MOCK] reg для:", req.email);

			// throw new ConnectError(
			// 	"Пользователь с таким email уже зарегистрирован",
			// 	Code.AlreadyExists,
			// );

			await new Promise((r) => setTimeout(r, 1600));

			if (req.email !== "admin@mail.ru" && req.password !== "admin") {
				throw new ConnectError(
					"Неверный логин или пароль!",
					Code.PermissionDenied,
				);
			}

			return {
				email: req.email,
				token: "mock-jwt-token-xyz",
				refreshToken: "mock-refresh-token-xyz",
				expiresIn: 3600n,
			};
		},
		async updateUser(req) {
			return {
				user: {
					id: req.id,
					email: req.email || "user@example.com",
					createdAt: undefined,
					updatedAt: undefined,
				},
			};
		},
		async deleteUser(req) {
			return { success: true };
		},
	});
});

const transport = createConnectTransport({
	baseUrl: PUBLIC_API_BASE_URL || "https://demo.connectrpc.com",
});

export const client = createClient(UserService, mockTransport);
