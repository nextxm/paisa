<script lang="ts">
  import { onMount, tick } from "svelte";
  import {
    ChevronDown,
    ChevronRight,
    FilePlus2,
    FileText,
    Folder,
    PanelLeftClose,
    PanelLeftOpen,
    PanelRightClose,
    PanelRightOpen,
    Save,
    ShieldCheck,
    Trash2
  } from "lucide-svelte";
  import { apiGet, authHeaders, runtime, triggerSync } from "$lib/api";

  interface LedgerFile {
    file_name: string;
    name?: string;
    content: string;
  }

  let {
    targetFile = "",
    targetLine = 0,
    jumpSeq = 0
  }: {
    targetFile?: string;
    targetLine?: number;
    jumpSeq?: number;
  } = $props();

  let files = $state<LedgerFile[]>([]);
  let selectedFileName = $state("");
  let content = $state("");
  let textareaEl = $state<HTMLTextAreaElement | null>(null);
  let newFileName = $state("");
  let loading = $state(true);
  let saving = $state(false);
  let validating = $state(false);
  let status = $state<string | null>(null);
  let error = $state<string | null>(null);
  let validationOutput = $state("");
  let validationErrors = $state<Array<{ line_from?: number; line?: number; message?: string }>>([]);

  let leftCollapsed = $state(false);
  let rightCollapsed = $state(false);
  let leftWidth = $state(240);
  let rightWidth = $state(360);
  let collapsedFolders = $state<Record<string, boolean>>({});
  let layoutEl = $state<HTMLDivElement | null>(null);

  interface TreeNode {
    name: string;
    path: string;
    isFolder: boolean;
    fileName?: string;
    children: TreeNode[];
  }

  const fileTree = $derived.by((): TreeNode[] => {
    const root: TreeNode[] = [];
    for (const f of files) {
      const parts = f.file_name.split(/[\\/]+/).filter(Boolean);
      let curList = root;
      let accPath = "";
      for (let i = 0; i < parts.length; i++) {
        const part = parts[i];
        accPath = accPath ? `${accPath}/${part}` : part;
        const isLast = i === parts.length - 1;
        if (isLast) {
          curList.push({
            name: part,
            path: accPath,
            isFolder: false,
            fileName: f.file_name,
            children: []
          });
        } else {
          let folder = curList.find((n) => n.isFolder && n.name === part);
          if (!folder) {
            folder = { name: part, path: accPath, isFolder: true, children: [] };
            curList.push(folder);
          }
          curList = folder.children;
        }
      }
    }
    const sortNodes = (nodes: TreeNode[]): TreeNode[] =>
      nodes
        .sort((a, b) =>
          a.isFolder === b.isFolder ? a.name.localeCompare(b.name) : a.isFolder ? -1 : 1
        )
        .map((n) => (n.isFolder ? { ...n, children: sortNodes(n.children) } : n));
    return sortNodes(root);
  });

  function startResizeLeft(e: MouseEvent) {
    e.preventDefault();
    const startX = e.clientX;
    const startW = leftWidth;
    const onMove = (ev: MouseEvent) => {
      leftWidth = Math.max(150, Math.min(480, startW + (ev.clientX - startX)));
    };
    const onUp = () => {
      window.removeEventListener("mousemove", onMove);
      window.removeEventListener("mouseup", onUp);
    };
    window.addEventListener("mousemove", onMove);
    window.addEventListener("mouseup", onUp);
  }

  function startResizeRight(e: MouseEvent) {
    e.preventDefault();
    const startX = e.clientX;
    const startW = rightWidth;
    const onMove = (ev: MouseEvent) => {
      rightWidth = Math.max(200, Math.min(640, startW - (ev.clientX - startX)));
    };
    const onUp = () => {
      window.removeEventListener("mousemove", onMove);
      window.removeEventListener("mouseup", onUp);
    };
    window.addEventListener("mousemove", onMove);
    window.addEventListener("mouseup", onUp);
  }

  async function loadFiles(selectName?: string) {
    const res = await apiGet<{ files?: LedgerFile[] }>("/api/editor/files");
    files = (res.files || []).map((f) => ({
      ...f,
      file_name: f.file_name || f.name || ""
    }));
    const target = selectName || targetFile || selectedFileName || files[0]?.file_name || "";
    if (target) {
      selectFile(target);
      if (targetFile && target === targetFile && targetLine > 0) {
        await focusLine(targetLine);
      }
    }
  }

  async function focusLine(lineNumber: number) {
    if (lineNumber <= 0) return;
    await tick();
    if (!textareaEl) return;
    const text = content;
    const lines = text.split("\n");
    const clampedLine = Math.min(Math.max(1, lineNumber), lines.length);
    let startOffset = 0;
    for (let i = 0; i < clampedLine - 1; i++) {
      startOffset += lines[i].length + 1;
    }
    const endOffset = startOffset + (lines[clampedLine - 1]?.length ?? 0);
    textareaEl.focus();
    textareaEl.setSelectionRange(startOffset, endOffset);

    const computed = window.getComputedStyle(textareaEl);
    const lineHeight = Number.parseFloat(computed.lineHeight) || 19.5;
    const targetScrollTop = Math.max(
      0,
      (clampedLine - 1) * lineHeight - textareaEl.clientHeight / 3
    );
    textareaEl.scrollTop = targetScrollTop;
  }

  $effect(() => {
    const seq = jumpSeq;
    const file = targetFile;
    const line = targetLine;
    if (seq > 0 && file && files.length > 0) {
      if (files.some((f) => f.file_name === file)) {
        selectFile(file);
        if (line > 0) {
          void focusLine(line);
        }
      }
    }
  });

  onMount(async () => {
    try {
      await loadFiles();
    } catch (e) {
      error = e instanceof Error ? e.message : "Failed to load ledger files";
    } finally {
      loading = false;
    }
  });

  function selectFile(name: string) {
    selectedFileName = name;
    const found = files.find((f) => f.file_name === name);
    content = found?.content || "";
    status = null;
    error = null;
    void validateFile();
  }

  async function validateFile() {
    if (!selectedFileName) return;
    validating = true;
    try {
      const res = await fetch("/api/editor/validate", {
        method: "POST",
        headers: { "Content-Type": "application/json", ...authHeaders() },
        body: JSON.stringify({ name: selectedFileName, content })
      });
      const data = (await res.json()) as {
        errors?: Array<{ line_from?: number; line?: number; message?: string }>;
        output?: string;
      };
      validationErrors = data.errors || [];
      validationOutput = data.output || "";
      if (validationErrors.length > 0) {
        error = validationErrors
          .map((e) => `Line ${e.line_from ?? e.line ?? "?"}: ${e.message ?? "Syntax error"}`)
          .join(" · ");
      } else {
        error = null;
        status = "Ledger syntax valid.";
      }
    } catch {
      // ignore transient validation network failure
    } finally {
      validating = false;
    }
  }

  async function saveFile() {
    if (!selectedFileName) return;
    saving = true;
    status = null;
    error = null;
    try {
      const res = await fetch("/api/editor/save", {
        method: "POST",
        headers: { "Content-Type": "application/json", ...authHeaders() },
        body: JSON.stringify({ name: selectedFileName, file_name: selectedFileName, content })
      });
      const data = (await res.json()) as {
        saved?: boolean;
        message?: string;
        errors?: Array<{ line_from?: number; line?: number; message?: string }>;
      };
      if (!res.ok || data.saved === false) {
        error = data.message || "Failed to save ledger file";
        if (data.errors) validationErrors = data.errors;
        return;
      }
      files = files.map((f) => (f.file_name === selectedFileName ? { ...f, content } : f));
      await triggerSync({ journal: true });
      await validateFile();
      status = "Saved and synced.";
    } catch (e) {
      error = e instanceof Error ? e.message : "Failed to save ledger file";
    } finally {
      saving = false;
    }
  }

  async function createFile() {
    const clean = newFileName.trim();
    if (!clean) return;
    const res = await fetch("/api/editor/save", {
      method: "POST",
      headers: { "Content-Type": "application/json", ...authHeaders() },
      body: JSON.stringify({ name: clean, file_name: clean, content: "", operation: "create" })
    });
    const data = (await res.json()) as { saved?: boolean; message?: string };
    if (data.saved) {
      newFileName = "";
      await loadFiles(clean);
      status = `Created ${clean}`;
    } else {
      error = data.message || "Failed to create file";
    }
  }

  async function deleteBackups() {
    if (!selectedFileName) return;
    const res = await fetch("/api/editor/file/delete_backups", {
      method: "POST",
      headers: { "Content-Type": "application/json", ...authHeaders() },
      body: JSON.stringify({ name: selectedFileName })
    });
    if (res.ok) status = "Backups deleted.";
  }
</script>

{#if loading}
  <div
    class="h-80 rounded-xl bg-[var(--surface)] border border-[var(--border)] animate-pulse"
  ></div>
{:else if files.length === 0}
  <div
    class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-8 text-center text-sm text-[var(--muted)]"
  >
    No ledger files found.
  </div>
{:else}
  <div
    class="rounded-xl bg-[var(--surface)] border border-[var(--border)] overflow-hidden flex flex-col lg:h-[calc(100vh-14.5rem)] lg:min-h-[600px]"
  >
    <!-- Top Toolbar -->
    <div
      class="px-4 py-2.5 border-b border-[var(--border)] flex flex-wrap items-center justify-between gap-2"
    >
      <div class="flex items-center gap-2">
        <button
          type="button"
          onclick={() => (leftCollapsed = !leftCollapsed)}
          title="Toggle File Browser"
          class="p-1.5 rounded-md border border-[var(--border)] text-[var(--muted)] hover:text-[var(--text)] cursor-pointer"
        >
          {#if leftCollapsed}
            <PanelLeftOpen class="w-3.5 h-3.5" />
          {:else}
            <PanelLeftClose class="w-3.5 h-3.5" />
          {/if}
        </button>
        <span class="text-xs font-semibold font-mono">{selectedFileName}</span>
        {#if status}
          <span class="text-xs text-[var(--gain)]">{status}</span>
        {/if}
        {#if validationErrors.length > 0}
          <span
            class="text-xs px-2 py-0.5 rounded bg-[var(--loss)]/15 text-[var(--loss)] font-medium"
          >
            {validationErrors.length} error(s)
          </span>
        {/if}
      </div>

      <div class="flex items-center gap-2">
        {#if !runtime.readonly}
          <button
            type="button"
            disabled={validating}
            class="inline-flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg text-xs font-medium border border-[var(--border)] hover:border-[var(--accent)] cursor-pointer"
            onclick={validateFile}
          >
            <ShieldCheck class="w-3.5 h-3.5" />
            <span>{validating ? "Checking…" : "Validate"}</span>
          </button>
          <button
            type="button"
            class="inline-flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg text-xs font-medium border border-[var(--border)] hover:border-[var(--loss)] text-[var(--muted)] hover:text-[var(--loss)] cursor-pointer"
            onclick={deleteBackups}
          >
            <Trash2 class="w-3.5 h-3.5" />
            <span>Clear Backups</span>
          </button>
          <button
            type="button"
            disabled={saving}
            class="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium bg-[var(--accent)] text-[var(--accent-fg)] hover:opacity-90 disabled:opacity-50 cursor-pointer"
            onclick={saveFile}
          >
            <Save class="w-3.5 h-3.5" />
            <span>{saving ? "Saving…" : "Save & Sync"}</span>
          </button>
        {/if}
        <button
          type="button"
          onclick={() => (rightCollapsed = !rightCollapsed)}
          title="Toggle Ledger/Beancount Summary"
          class="p-1.5 rounded-md border border-[var(--border)] text-[var(--muted)] hover:text-[var(--text)] cursor-pointer"
        >
          {#if rightCollapsed}
            <PanelRightOpen class="w-3.5 h-3.5" />
          {:else}
            <PanelRightClose class="w-3.5 h-3.5" />
          {/if}
        </button>
      </div>
    </div>

    {#if error}
      <div
        class="px-4 py-2 bg-[var(--loss)]/10 text-xs text-[var(--loss)] border-b border-[var(--border)]"
      >
        {error}
      </div>
    {/if}

    {#snippet treeNodes(nodes: TreeNode[], depth: number)}
      {#each nodes as node}
        {#if node.isFolder}
          {@const isClosed = Boolean(collapsedFolders[node.path])}
          <div>
            <button
              type="button"
              onclick={() => (collapsedFolders[node.path] = !isClosed)}
              style="padding-left: {depth * 12 + 6}px"
              class="w-full flex items-center gap-1.5 py-1 pr-2 rounded-md text-left hover:bg-[var(--surface-elevated)] cursor-pointer text-[var(--muted)] font-medium"
            >
              {#if isClosed}
                <ChevronRight class="w-3.5 h-3.5 shrink-0" />
              {:else}
                <ChevronDown class="w-3.5 h-3.5 shrink-0" />
              {/if}
              <Folder class="w-3.5 h-3.5 shrink-0" />
              <span class="truncate">{node.name}</span>
            </button>
            {#if !isClosed}
              {@render treeNodes(node.children, depth + 1)}
            {/if}
          </div>
        {:else}
          <button
            type="button"
            onclick={() => node.fileName && selectFile(node.fileName)}
            style="padding-left: {depth * 12 + 10}px"
            class="w-full flex items-center gap-1.5 py-1 pr-2 rounded-md text-left cursor-pointer truncate {selectedFileName ===
            node.fileName
              ? 'bg-[var(--accent)]/15 text-[var(--accent)] font-semibold'
              : 'hover:bg-[var(--surface-elevated)]'}"
          >
            <FileText class="w-3.5 h-3.5 shrink-0" />
            <span class="truncate">{node.name}</span>
          </button>
        {/if}
      {/each}
    {/snippet}

    <!-- 3-Part Resizable Editor Layout -->
    <div bind:this={layoutEl} class="flex-1 min-h-0 flex flex-col lg:flex-row">
      <!-- Left: Hierarchical File Browser -->
      {#if !leftCollapsed}
        <div
          style="width: {leftWidth}px"
          class="w-full lg:shrink-0 border-b lg:border-b-0 bg-[var(--surface-elevated)]/30 p-3 flex flex-col gap-3 min-h-0"
        >
          <div
            class="text-[11px] font-semibold uppercase tracking-wider text-[var(--muted)] shrink-0"
          >
            Files ({files.length})
          </div>
          {#if !runtime.readonly}
            <div class="flex gap-1.5 shrink-0">
              <input
                type="text"
                bind:value={newFileName}
                placeholder="new.ledger"
                class="flex-1 min-w-0 px-2 py-1 rounded text-xs bg-[var(--surface)] border border-[var(--border)] outline-none"
              />
              <button
                type="button"
                onclick={createFile}
                title="Create file"
                class="px-2 py-1 rounded bg-[var(--accent)] text-[var(--accent-fg)] text-xs cursor-pointer"
              >
                <FilePlus2 class="w-3.5 h-3.5" />
              </button>
            </div>
          {/if}
          <div
            class="space-y-0.5 max-h-[320px] lg:max-h-none lg:flex-1 lg:min-h-0 overflow-y-auto text-xs"
          >
            {@render treeNodes(fileTree, 0)}
          </div>
        </div>
        <button
          type="button"
          aria-label="Resize file browser"
          onmousedown={startResizeLeft}
          class="hidden lg:block w-1.5 shrink-0 bg-[var(--border)]/60 hover:bg-[var(--accent)] cursor-col-resize transition-colors"
        ></button>
      {/if}

      <!-- Middle: Editor -->
      <div class="flex-1 min-w-0 min-h-[420px] lg:min-h-0 flex flex-col">
        <textarea
          bind:this={textareaEl}
          bind:value={content}
          readonly={runtime.readonly}
          spellcheck="false"
          class="w-full flex-1 min-h-0 p-4 bg-transparent font-mono text-xs leading-relaxed outline-none resize-none"
        ></textarea>
      </div>

      <!-- Right: Ledger / Beancount Summary & Output -->
      {#if !rightCollapsed}
        <button
          type="button"
          aria-label="Resize summary panel"
          onmousedown={startResizeRight}
          class="hidden lg:block w-1.5 shrink-0 bg-[var(--border)]/60 hover:bg-[var(--accent)] cursor-col-resize transition-colors"
        ></button>
        <div
          style="width: {rightWidth}px"
          class="w-full lg:shrink-0 border-t lg:border-t-0 bg-[var(--surface-elevated)]/30 p-3 space-y-2.5 flex flex-col min-h-0"
        >
          <div class="flex items-center justify-between shrink-0">
            <span class="text-[11px] font-semibold uppercase tracking-wider text-[var(--muted)]">
              Ledger / Beancount Summary
            </span>
            {#if validating}
              <span class="text-[11px] text-[var(--muted)]">Running…</span>
            {/if}
          </div>
          {#if validationErrors.length > 0}
            <div class="space-y-1.5 text-xs shrink-0 max-h-40 overflow-y-auto">
              {#each validationErrors as ve}
                <div
                  class="p-2 rounded bg-[var(--loss)]/10 border border-[var(--loss)]/30 text-[var(--loss)]"
                >
                  Line {ve.line_from ?? ve.line ?? "?"}: {ve.message}
                </div>
              {/each}
            </div>
          {/if}
          {#if validationOutput}
            <pre
              class="flex-1 min-h-0 p-3 rounded-lg bg-[var(--surface)] border border-[var(--border)] font-mono text-[11px] leading-relaxed overflow-auto max-h-[360px] lg:max-h-none whitespace-pre-wrap">{validationOutput}</pre>
          {:else}
            <div
              class="text-xs text-[var(--muted)] p-3 rounded-lg bg-[var(--surface)] border border-[var(--border)]"
            >
              Click <strong>Validate</strong> or <strong>Save &amp; Sync</strong> to view ledger/beancount
              summary output.
            </div>
          {/if}
        </div>
      {/if}
    </div>
  </div>
{/if}
