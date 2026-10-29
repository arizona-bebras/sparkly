import { z } from 'zod';

export const formSchema = z
  .object({
    email: z.email('Некорректный email'),
    password: z
      .string()
      .min(6, 'Пароль должен содержать не менее 6 символов')
      .max(50, 'Пароль не должен превышать 50 символов'),
    confirmPassword: z.string().min(1, 'Подтвердите пароль'),
  })
  .refine((data) => data.password === data.confirmPassword, {
    message: 'Пароли не совпадают',
    path: ['confirmPassword'],
  });

export type FormSchema = typeof formSchema;
