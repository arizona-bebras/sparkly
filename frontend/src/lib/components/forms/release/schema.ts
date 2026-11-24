import { z } from 'zod';

export const formSchema = z.object({
  version: z
    .string()
    .min(1, 'Версия должна содержать не менее 1 символа')
    .max(30, 'Версия не должна превышать 30 символов'),
  name: z
    .string()
    .min(1, 'Название должно содержать не менее 1 символа')
    .max(100, 'Название не должно превышать 100 символов'),
});

export type FormSchema = typeof formSchema;
