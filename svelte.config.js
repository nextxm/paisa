import adapter from "@sveltejs/adapter-static";
import { vitePreprocess } from "@sveltejs/vite-plugin-svelte";

const base = process.env.PAISA_V1_BASE ?? "/v1";
const outDir = base ? `web/static${base}` : "web/static";

/** @type {import('@sveltejs/kit').Config} */
const config = {
  // Consult https://kit.svelte.dev/docs/integrations#preprocessors
  // for more information about preprocessors
  preprocess: vitePreprocess(),

  onwarn: (warning, handler) => {
    if (warning.code.startsWith("a11y-")) return;
    handler(warning);
  },

  kit: {
    paths: {
      base
    },
    adapter: adapter({
      pages: outDir,
      assets: outDir,
      out: outDir,
      fallback: "index.html"
    })
  }
};

export default config;
