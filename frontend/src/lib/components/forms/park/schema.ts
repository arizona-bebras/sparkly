import { z } from "zod";

export const formSchema = z.object({
	name: z
		.string()
		.min(2, "Название должно содержать не менее 2 символов")
		.max(50, "Название не должно превышать 50 символов"),
	description: z
		.string()
		.max(255, "Описание не должно превышать 255 символов")
		.default(""),
});

export type FormSchema = typeof formSchema;

