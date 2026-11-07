import { z } from 'zod';

export const formSchema = z.object({
  name: z
    .string()
    .min(1, 'Название должно содержать не менее 1 символа')
    .max(30, 'Название не должно превышать 30 символов'),
  description: z
    .string()
    .min(1, 'Описание должно содержать не менее 1 символа')
    .max(255, 'Описание не должно превышать 255 символов'),
});

export type FormSchema = typeof formSchema;
