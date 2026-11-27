import { sveltekit } from "@sveltejs/kit/vite";
import tailwindcss from "@tailwindcss/vite";
import { execSync } from "node:child_process";
import fs from "node:fs";
import path from "node:path";
import { defineConfig } from "vite";

function resolveBuildInfo() {
  let commitHash = "";
  let branch = "";
  let tag = "";
  let appVersion = "dev";

  try {
    commitHash = execSync("git rev-parse --short HEAD", { stdio: "pipe" }).toString().trim();
    branch = execSync("git rev-parse --abbrev-ref HEAD", { stdio: "pipe" }).toString().trim();
    try {
      tag = execSync("git describe --tags --exact-match", { stdio: "pipe" }).toString().trim();
    } catch {}
    appVersion = execSync("git describe --tags --always --dirty", { stdio: "pipe" })
      .toString()
      .trim();
  } catch {
    const repoRoot = path.resolve(process.cwd(), "..");
    try {
      const headContent = fs.readFileSync(path.join(repoRoot, ".git/HEAD"), "utf-8").trim();
      if (headContent.startsWith("ref: ")) {
        const refPath = headContent.slice(5);
        branch = refPath.replace("refs/heads/", "");
        try {
          commitHash = fs
            .readFileSync(path.join(repoRoot, ".git", refPath), "utf-8")
            .trim()
            .slice(0, 7);
        } catch {
          try {
            const packed = fs.readFileSync(path.join(repoRoot, ".git/packed-refs"), "utf-8");
            const line = packed.split("\n").find((l) => l.endsWith(refPath));
            if (line) commitHash = line.split(" ")[0].slice(0, 7);
          } catch {}
        }
      } else {
        commitHash = headContent.slice(0, 7);
        branch = "HEAD";
      }
    } catch {}

    try {
      const versionGo = fs.readFileSync(path.join(repoRoot, "cmd/version.go"), "utf-8");
      const match = versionGo.match(/var Version = "([^"]+)"/);
      if (match) appVersion = match[1];
    } catch {}
  }

  return {
    version: appVersion,
    commitHash,
    branch,
    tag,
    buildDate: new Date().toISOString()
  };
}

export default defineConfig({
  define: {
    __BUILD_INFO__: JSON.stringify(resolveBuildInfo())
  },
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
