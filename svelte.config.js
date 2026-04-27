import adapter from "@sveltejs/adapter-static";
import { vitePreprocess } from "@sveltejs/vite-plugin-svelte";

/** @type {import('@sveltejs/kit').Config} */
const config = {
  // Consult https://kit.svelte.dev/docs/integrations#preprocessors
  // for more information about preprocessors
  preprocess: vitePreprocess(),

  onwarn: (warning, handler) => {
    if (warning.code.startsWith("a11y-")) return;
    handler(warning);
  },

  // Keep Svelte 4 component API so existing code continues to work while
  // components are incrementally migrated to Svelte 5 runes.
  compilerOptions: {
    compatibility: {
      componentApi: 4
    }
  },

  kit: {
    adapter: adapter({
      pages: "web/static",
      assets: "web/static",
      out: "web/static",
      fallback: "index.html"
    })
  }
};

export default config;
