import {
	defaults,
	type FormOptions,
	type SuperForm,
	superForm,
} from "sveltekit-superforms";
import { zod4 } from "sveltekit-superforms/adapters";
import type { SuperFormData } from "sveltekit-superforms/client";
import type { validateAsync, z } from "zod";

export type SpaFormReturn<T extends Record<string, unknown>, M = any> = Omit<
	SuperForm<T, M>,
	"form"
> & {
	form: SuperForm<T, M>;
	formData: SuperFormData<T>;
	isValid: () => boolean;
};

export let isValid = $state({ valid: false });

export function spaForm<
	T extends Record<string, unknown> = Record<string, unknown>,
	M = any,
>(formSchema: z.ZodType<T>, options?: FormOptions<T, M>): SpaFormReturn<T, M> {
	let isValid = $state({ valid: false });
	const form = superForm<T, M>(defaults(zod4(formSchema) as any), {
		SPA: true,
		validators: zod4(formSchema) as any,
		onUpdated({ form }) {
			console.log("Форма отправлена!. Её данные:", form.data);
		},
		onSubmit() {
			isValid.valid = false;
		},
		async onChange() {
			const validate = await form.validateForm();
			isValid.valid = validate.valid;
		},
		...options,
	});

	return {
		...form,
		form,
		formData: form.form,
		isValid: () => isValid.valid,
	};
}
