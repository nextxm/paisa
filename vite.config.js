import { sveltekit } from "@sveltejs/kit/vite";
import { nodePolyfills } from "vite-plugin-node-polyfills";
import { VitePWA } from "vite-plugin-pwa";
import { execSync } from "child_process";
import fs from "fs";

let commitHash = "";
let branch = "";
let tag = "";
let appVersion = "dev";

try {
  commitHash = execSync("git rev-parse --short HEAD").toString().trim();
  branch = execSync("git rev-parse --abbrev-ref HEAD").toString().trim();
  try {
    tag = execSync("git describe --tags --exact-match").toString().trim();
  } catch (e) { }
  // Use the same logic as Makefile for the main version string
  appVersion = execSync("git describe --tags --always --dirty").toString().trim();
} catch (e) {
  // Fallback to cmd/version.go if git fails
  try {
    const versionGo = fs.readFileSync("cmd/version.go", "utf-8");
    const match = versionGo.match(/var Version = \"([^\"]+)\"/);
    if (match) {
      appVersion = match[1];
    }
  } catch (e) { }
}

const buildDate = new Date().toISOString();

/** @type {import('vite').UserConfig} */
const config = {
  define: {
    __BUILD_INFO__: JSON.stringify({
      version: appVersion,
      commitHash,
      branch,
      tag,
      buildDate
    })
  },
  build: {
    target: "es2021"
  },
  plugins: [
    sveltekit(),
    nodePolyfills({
      globals: {
        Buffer: true
      }
    }),
    VitePWA({
      registerType: "prompt",
      manifest: {
        name: "Paisa – Personal Finance Manager",
        short_name: "Paisa",
        description:
          "Open source personal finance manager built on top of the ledger double-entry accounting tool.",
        theme_color: "#1e1e2e",
        background_color: "#1e1e2e",
        display: "standalone",
        start_url: "/",
        icons: [
          {
            src: "pwa-192x192.png",
            sizes: "192x192",
            type: "image/png"
          },
          {
            src: "pwa-512x512.png",
            sizes: "512x512",
            type: "image/png"
          },
          {
            src: "pwa-maskable-512x512.png",
            sizes: "512x512",
            type: "image/png",
            purpose: "maskable"
          }
        ]
      },
      workbox: {
        globPatterns: ["**/*.{js,css,html,ico,png,svg,woff}"],
        maximumFileSizeToCacheInBytes: 3 * 1024 * 1024,
        navigateFallback: "/",
        navigateFallbackAllowlist: [/^(?!\/_app\/immutable).*$/],
        runtimeCaching: [
          {
            urlPattern: /\.woff2$/i,
            handler: "CacheFirst",
            options: {
              cacheName: "font-cache",
              expiration: {
                maxEntries: 20,
                maxAgeSeconds: 365 * 24 * 60 * 60
              },
              cacheableResponse: {
                statuses: [0, 200]
              }
            }
          }
        ]
      },
      devOptions: {
        enabled: true
      }
    })
  ],
  server: {
    proxy: {
      "/api": {
        target: "http://localhost:7500"
      }
    },
    fs: {
      allow: ["./fonts"]
    }
  }
};

export default config;
