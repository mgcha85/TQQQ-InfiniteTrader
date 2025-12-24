import flowbite from 'flowbite/plugin';
import daisyui from 'daisyui';
import type { Config } from 'tailwindcss';

export default {
  content: [
    './src/**/*.{html,js,svelte,ts}',
    './node_modules/flowbite-svelte/**/*.{html,js,svelte,ts}'
  ],
  theme: {
    extend: {}
  },
  plugins: [
    flowbite,
    daisyui
  ],
  daisyui: {
    themes: ["light", "dark", "corporate", "business"],
  }
} as Config;
