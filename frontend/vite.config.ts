import { sveltekit } from "@sveltejs/kit/vite";
import tailwindcss from "@tailwindcss/vite";
import { defineConfig } from "vite";

export default defineConfig({
  plugins: [tailwindcss(), sveltekit()],
  css: {
    postcss: {
      plugins: []
    }
  },
  server: {
    proxy: {
      "/api": {
        target: "http://localhost:7500",
        changeOrigin: true
      },
      "/connect": {
        target: "http://localhost:7500",
        changeOrigin: true
      }
    }
  }
});
