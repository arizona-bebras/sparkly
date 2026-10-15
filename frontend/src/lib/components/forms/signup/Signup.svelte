<script lang="ts">
import { createMutation } from "@tanstack/svelte-query";
import type { HTMLAttributes } from "svelte/elements";
import { goto } from "$app/navigation";
import { formSchema } from "$lib/components/forms/signup/schema";
import { Button } from "$lib/components/ui/button/index.js";
import * as Card from "$lib/components/ui/card/index.js";
import * as Field from "$lib/components/ui/field/index.js";
import * as Form from "$lib/components/ui/form/index.js";
import { Input } from "$lib/components/ui/input/index.js";
import { spaForm } from "$lib/form-spa-util.svelte";
import { cn } from "$lib/utils.js";
import { client } from "$proto";
import SubmitBtn from "../SubmitBtn.svelte";

let { class: className, ...restProps }: HTMLAttributes<HTMLDivElement> =
	$props();

const { form, formData, isValid, enhance } = spaForm(formSchema, {
	async onUpdated({ form }) {
		await authMutation.mutateAsync({
			email: form.data.email,
			password: form.data.password,
		});
	},
});

const authMutation = createMutation(() => ({
	mutationKey: ["auth"],
	mutationFn: async (form: { email: string; password: string }) =>
		await client.user.reg({
			email: form.email,
			password: form.password,
		}),
	onSuccess: () => {
		goto("/");
	},
}));
</script>

<div class={cn("flex flex-col gap-6", className)} {...restProps}>
	<Card.Root class="animate-in fade-in zoom-in-95 slide-in-from-bottom-4 duration-600 ease-out shadow-lg border-border/80">
		<Card.Header class="text-center animate-in fade-in slide-in-from-top-2 duration-500 delay-100 ease-out fill-mode-both">
			<Card.Title class="text-xl">Создайте свой аккаунт</Card.Title>
			<Card.Description>Заполните поля ниже, чтобы создать свой аккаунт</Card.Description>
		</Card.Header>
		<Card.Content>
			<form method="POST" use:enhance class="animate-in fade-in slide-in-from-bottom-2 duration-500 delay-200 ease-out fill-mode-both">
				<Field.Group>
					<Form.Field {form} name="email">
						<Form.Control>
						{#snippet children({ props })}
							<Input type="magic" magicProps={{type: "email", label: "Email", "placeholder": "Enter your email"}} {...props} bind:value={$formData.email} />
						{/snippet}
						</Form.Control>
						<Form.Description>Это ваша электронная почта.</Form.Description>
						<Form.FieldErrors />
					</Form.Field>
					<Form.Field {form} name="password">
						<Form.Control>
						{#snippet children({ props })}
							<Input type="magic" magicProps={{type: "password", label: "Password", "placeholder": "Create a password"}} {...props} bind:value={$formData.password} />
						{/snippet}
						</Form.Control>
						<Form.Description>Должен содержать не менее 6 символов.</Form.Description>
						<Form.FieldErrors />
					</Form.Field>
					<Field.Field>
						<SubmitBtn className="-mt-2.5" {isValid} isAuthError={authMutation.isError} isLoading={authMutation.isPending}  />
						<Field.Description class="text-center">
							Уже есть аккаунт? <a href="/auth/signin">Войти</a>
						</Field.Description>
					</Field.Field>
				</Field.Group>
			</form>
		</Card.Content>
	</Card.Root>
	<!-- <Field.Description class="px-6 text-center animate-in fade-in duration-500 delay-300 ease-out fill-mode-both">
		By clicking continue, you agree to our <a href="#/">Terms of Service</a>
		and <a href="#/">Privacy Policy</a>.
	</Field.Description> -->
</div>
