// import { redirect } from "@sveltejs/kit";

// export const handle = async ({ event, resolve }) => {
// 	const token = event.cookies.get("token");
// 	const isAuthPage = event.url.pathname.startsWith("/auth");

// 	if (!token && !isAuthPage) {
// 		redirect(307, "/auth/signin");
// 	}
// 	return resolve(event);
// };
