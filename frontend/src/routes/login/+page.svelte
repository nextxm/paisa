<script lang="ts">
  import { goto } from "$app/navigation";
  import { appPath } from "$lib/api";
  import Logo from "$lib/components/Logo.svelte";

  let username = $state("");
  let password = $state("");
  let loading = $state(false);
  let error = $state<string | null>(null);

  async function submit(e: SubmitEvent) {
    e.preventDefault();
    loading = true;
    error = null;
    try {
      const res = await fetch("/api/auth/login", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ username, password })
      });
      if (!res.ok) {
        error = "Invalid credentials";
        return;
      }
      const data = (await res.json()) as { token?: string };
      if (data.token) {
        try {
          localStorage.setItem("token", data.token);
        } catch {}
      }
      await goto(appPath("/"));
    } catch (err) {
      error = err instanceof Error ? err.message : "Login failed";
    } finally {
      loading = false;
    }
  }
</script>

<div class="min-h-[75vh] flex items-center justify-center p-4">
  <div
    class="w-full max-w-sm rounded-xl bg-[var(--surface)] border border-[var(--border)] p-6 space-y-5"
  >
    <div class="flex items-center gap-2.5">
      <Logo size={28} />
      <div>
        <div class="flex items-center gap-2">
          <h1 class="text-lg font-semibold leading-tight">Sign in to Paisa</h1>
          <span
            class="px-1.5 py-0.5 rounded-full text-[10px] font-mono font-semibold bg-[var(--accent)]/15 text-[var(--accent)]"
          >
            v2
          </span>
        </div>
        <p class="text-xs text-[var(--muted)]">Personal Finance Manager</p>
      </div>
    </div>

    {#if error}
      <div
        class="rounded-lg bg-[var(--loss)]/10 border border-[var(--loss)]/30 px-3 py-2 text-xs text-[var(--loss)]"
      >
        {error}
      </div>
    {/if}

    <form class="space-y-3.5 text-sm" onsubmit={submit}>
      <div>
        <label class="block text-xs font-medium text-[var(--muted)] mb-1" for="login-user">
          Username
        </label>
        <input
          id="login-user"
          type="text"
          bind:value={username}
          class="w-full px-3 py-2 rounded-lg bg-[var(--surface-elevated)] border border-[var(--border)] outline-none"
          required
        />
      </div>
      <div>
        <label class="block text-xs font-medium text-[var(--muted)] mb-1" for="login-pass">
          Password
        </label>
        <input
          id="login-pass"
          type="password"
          bind:value={password}
          class="w-full px-3 py-2 rounded-lg bg-[var(--surface-elevated)] border border-[var(--border)] outline-none"
          required
        />
      </div>
      <button
        type="submit"
        disabled={loading}
        class="w-full py-2 rounded-lg text-sm font-medium bg-[var(--accent)] text-[var(--accent-fg)] hover:opacity-90 disabled:opacity-50 cursor-pointer"
      >
        {loading ? "Signing in…" : "Sign in"}
      </button>
    </form>
  </div>
</div>
