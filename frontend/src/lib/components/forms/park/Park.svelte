<script lang="ts">
import { createMutation } from "@tanstack/svelte-query";
import { onMount } from "svelte";
import type { HTMLAttributes } from "svelte/elements";
import SubmitBtn from "$lib/components/forms/SubmitBtn.svelte";
import { FieldGroup } from "$lib/components/ui/field/index.js";
import * as Form from "$lib/components/ui/form/index.js";
import { Input } from "$lib/components/ui/input/index.js";
import { spaForm } from "$lib/form-spa-util.svelte";
import { cn } from "$lib/utils.js";
import { client } from "$proto";
import type { Park } from "../../../../../proto/gen/backend/proto/service_pb.js";
import { formSchema } from "./schema.js";

let {
	class: className,
	onSuccess,
	park,
	...restProps
}: HTMLAttributes<HTMLFormElement> & {
	onSuccess?: (park?: Park) => void;
	park?: Park;
} = $props();

const createParkMutation = createMutation(() => ({
	mutationKey: ["createPark"],
	mutationFn: async (data: { name: string; description: string }) => {
		return await client.park.createPark({
			name: data.name,
			description: data.description,
		});
	},
	onSuccess: (response) => { onSuccess?.(response.park); },
}));

const updateParkMutation = createMutation(() => ({
	mutationKey: ["updatePark"],
	mutationFn: async (data: { name: string; description: string }) => {
		return await client.park.updatePark({
			id: park?.id,
			name: data.name,
			description: data.description,
		});
	},
	onSuccess: (response) => { onSuccess?.(response.park); },
}));

let currentAction = $derived(park ? updateParkMutation : createParkMutation);

const { form, formData, isValid, enhance } = spaForm(formSchema, {
	async onUpdated({ form }) {
		if (form.valid) {
			await currentAction.mutateAsync({
				name: form.data.name,
				description: form.data.description,
			});
		}
	},
});

onMount(() => {
	if (park) {
		$formData.name = park.name;
		$formData.description = park.description;
	}
});
</script>

<form
	method="POST"
	use:enhance
	class={cn("flex flex-col gap-4", className)}
	{...restProps}
>
	<FieldGroup>
		<Form.Field {form} name="name">
			<Form.Control>
				{#snippet children({ props })}
					<Input
						type="magic"
						magicProps={{ type: "text", label: "Название", placeholder: "Введите название парка" }}
						{...props}
						bind:value={$formData.name}
					/>
				{/snippet}
			</Form.Control>
			<Form.Description>Название вашего парка или площадки.</Form.Description>
			<Form.FieldErrors />
		</Form.Field>

		<Form.Field {form} name="description">
			<Form.Control>
				{#snippet children({ props })}
					<Input
						type="magic"
						magicProps={{ type: "text", label: "Описание", placeholder: "Краткое описание парка" }}
						{...props}
						bind:value={$formData.description}
					/>
				{/snippet}
			</Form.Control>
			<Form.Description>Необязательное описание назначения парка.</Form.Description>
			<Form.FieldErrors />
		</Form.Field>

		<SubmitBtn
			{isValid}
			label={currentAction === createParkMutation ? "Создать парк": "Обновить парк"}
			isAuthError={currentAction.isError}
			isLoading={currentAction.isPending}
			errorMessage="Не удалось создать парк. Попробуйте снова."
		/>
	</FieldGroup>
</form>

