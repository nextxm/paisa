import adapter from "@sveltejs/adapter-static";
import { vitePreprocess } from "@sveltejs/vite-plugin-svelte";

const outDir = process.env.PAISA_EMBED === "1" ? "../web/static" : "build";

/** @type {import('@sveltejs/kit').Config} */
const config = {
  preprocess: vitePreprocess(),
  kit: {
    adapter: adapter({
      pages: outDir,
      assets: outDir,
      fallback: "index.html"
    })
  }
};

export default config;
