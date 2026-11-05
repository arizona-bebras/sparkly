import adapter from '@sveltejs/adapter-node';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';

/** @type {import('@sveltejs/kit').Config} */
const config = {
  preprocess: vitePreprocess(),
  kit: {
    adapter: adapter(),
    alias: {
      $proto: './proto/client.ts',
      '@/*': './src/lib/*',
    },
  },
  vitePlugin: {
    inspector: {
      toggleKeyCombo: 'alt-x',
      holdMode: false,
      showToggleButton: 'always',
      toggleButtonPos: 'bottom-right',
    },
  },
};

export default config;
