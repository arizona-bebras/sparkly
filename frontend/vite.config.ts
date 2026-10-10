import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig, lazyPlugins } from 'vite-plus';

export default defineConfig({
  plugins: lazyPlugins(() => [sveltekit()]),
  lint: {
    ignorePatterns: ['.svelte-kit/**', 'build/**', 'dist/**'],
  },
  fmt: {
    singleQuote: true,
    ignorePatterns: ['.svelte-kit/**', 'build/**', 'dist/**'],
  },
});
