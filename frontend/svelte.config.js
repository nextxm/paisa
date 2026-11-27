import adapter from "@sveltejs/adapter-static";
import { vitePreprocess } from "@sveltejs/vite-plugin-svelte";

const base = process.env.PAISA_BASE || "";
const outDir =
  process.env.PAISA_EMBED === "1" ? (base === "/v2" ? "../web/static/v2" : "../web/static") : "build";

/** @type {import('@sveltejs/kit').Config} */
const config = {
  preprocess: vitePreprocess(),
  kit: {
    paths: {
      base
    },
    adapter: adapter({
      pages: outDir,
      assets: outDir,
      fallback: "index.html"
    })
  }
};

export default config;
