import { sveltekit } from "@sveltejs/kit/vite";
import tailwindcss from "@tailwindcss/vite";
import { defineConfig, lazyPlugins } from "vite-plus";

export default defineConfig({
	plugins: lazyPlugins(() => [sveltekit(), tailwindcss()]),
	lint: {
		ignorePatterns: [".svelte-kit/**", "build/**", "dist/**"],
	},
	fmt: {
		singleQuote: true,
		ignorePatterns: [".svelte-kit/**", "build/**", "dist/**"],
	},
});
