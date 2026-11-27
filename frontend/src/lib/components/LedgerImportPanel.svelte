<script lang="ts">
  import { onMount } from "svelte";
  import {
    Check,
    Code2,
    Columns3,
    Copy,
    FilePlus2,
    Save,
    SlidersHorizontal,
    Sparkles,
    Trash2,
    Upload,
    WandSparkles
  } from "lucide-svelte";
  import { apiGet, authHeaders, formatMoney, runtime, triggerSync } from "$lib/api";
  import {
    asRows,
    columnName,
    extractShortNote,
    findMatch,
    generateLedgerText,
    parseCSVText,
    renderTemplatePreview,
    toCSVContent,
    type AccountTfIdfData,
    type ColumnMapping,
    type ImportPreset,
    type ImportPreviewRow,
    type ImportRule,
    type ImportTemplate,
    type StagedTransaction
  } from "$lib/import_helpers";

  type ImportMode = "smart" | "template";
  let importMode = $state<ImportMode>("smart");

  // Shared metadata
  let accountTfIdf = $state<AccountTfIdfData | null>(null);
  let templates = $state<ImportTemplate[]>([]);
  let presets = $state<ImportPreset[]>([]);
  let ledgerFiles = $state<string[]>([]);

  // File & tabular state
  let uploadedFileName = $state("");
  let rawFileContent = $state("");
  let detectedFormat = $state<"csv" | "ofx" | "qfx">("csv");
  let delimiter = $state(",");
  let tableData = $state<string[][]>([]);
  let tableRows = $state<Array<Record<string, string | number>>>([]);
  let columnCount = $state(0);
  let parseError = $state<string | null>(null);
  let statusMessage = $state<string | null>(null);

  // ---- Smart Ingestion State ----
  let baseAccount = $state("");
  let smartLoading = $state(false);
  let smartCommitting = $state(false);
  let stagedTransactions = $state<StagedTransaction[]>([]);
  let includedStaged = $state<boolean[]>([]);
  let stageFilter = $state<"all" | "ready" | "unassigned" | "duplicate">("all");
  let bulkAccount = $state("");
  let groupByDate = $state(true);

  // Column Mapper modal
  let columnMapping = $state<ColumnMapping | null>(null);
  let showColumnMapper = $state(false);
  let mapperAmountMode = $state<"split" | "single">("split");
  let mapperDraft = $state<ColumnMapping>({
    header_row: 0,
    date: 0,
    description: -1,
    debit: -1,
    credit: -1,
    amount: -1,
    dr_cr: -1,
    currency: -1
  });

  // Rule Studio modal
  let ruleStudioOpen = $state(false);
  let ruleTab = $state<"list" | "edit">("list");
  let rules = $state<ImportRule[]>([]);
  let rulesLoading = $state(false);
  let editingRule = $state<ImportRule>({
    name: "",
    priority: 100,
    enabled: true,
    payee_pattern: "",
    payee_match_type: "contains",
    memo_pattern: "",
    memo_match_type: "contains",
    tx_type: "any",
    target_account: "Expenses:Misc",
    tags: [],
    flag_for_review: false
  });
  let rawRuleTags = $state("");

  // Review Generated Ledger modal
  let reviewLedgerOpen = $state(false);
  let reviewLedgerContent = $state("");

  // ---- Classic Template Editor State ----
  let selectedTemplateName = $state("");
  let templateDraft = $state("");
  let saveTemplateAsName = $state("");
  let templateSaveModalOpen = $state(false);
  let selectedPresetName = $state("");
  let reverseRows = $state(false);
  let trimRows = $state(true);
  let previewRows = $state<ImportPreviewRow[]>([]);
  let includedPreviewRows = $state<boolean[]>([]);
  let renderedPreview = $state("");
  let previewRenderError = $state<string | null>(null);
  let saveToFileOpen = $state(false);
  let destinationFile = $state("");
  let savingToFile = $state(false);
  let newPresetName = $state("");

  const selectedTemplate = $derived(templates.find((t) => t.name === selectedTemplateName) ?? null);
  const selectedPreset = $derived(presets.find((p) => p.name === selectedPresetName) ?? null);

  const statementHeaderContext = $derived.by(() => {
    const parts: string[] = [];
    if (uploadedFileName) parts.push(uploadedFileName);
    if (rawFileContent) {
      const lines = rawFileContent.split(/\r?\n/);
      parts.push(...lines.slice(0, 100));
      if (lines.length > 100) parts.push(...lines.slice(-50));
    }
    for (let i = 0; i < Math.min(50, tableData.length); i++) {
      parts.push(tableData[i].join(" "));
    }
    return parts.join(" ").toLowerCase();
  });

  const orderedBaseAccounts = $derived.by(() => {
    const sourceCandidates = runtime.accounts.filter(
      (a) => a.startsWith("Assets:") || a.startsWith("Liabilities:")
    );
    const otherAccounts = runtime.accounts.filter(
      (a) => !a.startsWith("Assets:") && !a.startsWith("Liabilities:")
    );
    const all = [...sourceCandidates, ...otherAccounts];
    const header = statementHeaderContext;
    if (!header) return all;

    return [...all].sort((a, b) => {
      const scoreAccount = (acc: string) => {
        let score = 0;
        const tokens = acc
          .toLowerCase()
          .split(/[:\s_-]+/)
          .filter((t) => t.length >= 3 && t !== "assets" && t !== "liabilities");
        for (const tok of tokens) {
          if (header.includes(tok)) score += 10;
        }
        const leaf = acc.split(":").pop()?.toLowerCase();
        if (leaf && leaf.length >= 3 && header.includes(leaf)) score += 25;
        return score;
      };
      return scoreAccount(b) - scoreAccount(a);
    });
  });

  const topSuggestedBaseAccount = $derived.by(() => {
    if (!statementHeaderContext) return "";
    if (orderedBaseAccounts.length > 0) {
      const first = orderedBaseAccounts[0];
      const tokens = first
        .toLowerCase()
        .split(/[:\s_-]+/)
        .filter((t) => t.length >= 3 && t !== "assets" && t !== "liabilities");
      if (tokens.some((t) => statementHeaderContext.includes(t))) return first;
    }
    if (statementHeaderContext.includes("kotak") || statementHeaderContext.includes("811")) {
      return "Assets:Checking:Kotak";
    }
    if (statementHeaderContext.includes("icici")) {
      return statementHeaderContext.includes("credit card")
        ? "Liabilities:CreditCard:ICICI"
        : "Assets:Checking:ICICI";
    }
    if (statementHeaderContext.includes("axis")) return "Assets:Checking:Axis";
    if (statementHeaderContext.includes("hdfc")) return "Assets:Checking:HDFC";
    if (statementHeaderContext.includes("sbi")) return "Assets:Checking:SBI";
    return "";
  });

  $effect(() => {
    if (!baseAccount && topSuggestedBaseAccount) {
      baseAccount = topSuggestedBaseAccount;
    }
  });

  const mapperHeaderColumns = $derived.by(() => {
    const row = tableData[mapperDraft.header_row] || [];
    const len = Math.max(row.length, columnCount);
    return Array.from({ length: len }, (_, idx) => {
      const label = String(row[idx] ?? "").trim();
      return {
        index: idx,
        label: label ? `Col ${idx + 1}: ${label}` : `Col ${idx + 1} (empty)`
      };
    });
  });

  const filteredStagedIndices = $derived.by(() =>
    stagedTransactions
      .map((tx, idx) => ({ tx, idx }))
      .filter(({ tx }) => {
        if (stageFilter === "duplicate") return Boolean(tx.is_duplicate);
        if (stageFilter === "unassigned") return !tx.selected_account;
        if (stageFilter === "ready") return !tx.is_duplicate && Boolean(tx.selected_account);
        return true;
      })
      .map(({ idx }) => idx)
  );

  const selectedStagedCount = $derived(stagedTransactions.filter((_, i) => includedStaged[i]).length);
  const selectedPreviewCount = $derived(previewRows.filter((_, i) => includedPreviewRows[i]).length);
  const previewColumns = $derived.by(() => {
    const keys = new Set<string>();
    for (const r of previewRows) {
      for (const k of Object.keys(r.row || {})) {
        if (k !== "index") keys.add(k);
      }
    }
    return Array.from(keys).sort((a, b) => a.length - b.length || a.localeCompare(b));
  });

  $effect(() => {
    if (tableRows.length === 0 || !templateDraft.trim()) {
      renderedPreview = "";
      previewRenderError = null;
      return;
    }
    try {
      const selectedRows =
        previewRows.length === tableRows.length
          ? tableRows.filter((_, i) => includedPreviewRows[i])
          : tableRows;
      renderedPreview = renderTemplatePreview(
        templateDraft,
        selectedRows,
        { reverse: reverseRows, trim: trimRows },
        accountTfIdf
      );
      previewRenderError = null;
    } catch (e) {
      previewRenderError = e instanceof Error ? e.message : "Template render error";
    }
  });

  onMount(async () => {
    const [tfIdfRes, tplRes, presetRes, filesRes] = await Promise.all([
      apiGet<AccountTfIdfData>("/api/account/tf_idf").catch(() => null),
      apiGet<{ templates?: ImportTemplate[] }>("/api/templates").catch(() => ({ templates: [] })),
      apiGet<{ presets?: ImportPreset[] }>("/api/import/presets").catch(() => ({ presets: [] })),
      apiGet<{ files?: Array<{ file_name?: string; name?: string }> }>("/api/editor/files").catch(
        () => ({ files: [] })
      )
    ]);
    accountTfIdf = tfIdfRes;
    templates = tplRes.templates || [];
    presets = presetRes.presets || [];
    ledgerFiles = (filesRes.files || [])
      .map((f) => f.file_name || f.name || "")
      .filter(Boolean);
    if (ledgerFiles.length > 0) destinationFile = ledgerFiles[0];
    if (templates.length > 0) {
      selectedTemplateName = templates[0].name;
      templateDraft = templates[0].content || "";
      saveTemplateAsName = templates[0].name;
    }
    const generic = presets.find((p) => p.name === "Generic Bank CSV") || presets[0];
    if (generic) {
      selectedPresetName = generic.name;
      if (generic.delimiter) delimiter = generic.delimiter;
    }
  });

  async function handleFileSelect(e: Event) {
    const input = e.currentTarget as HTMLInputElement;
    const file = input.files?.[0];
    if (!file) return;
    uploadedFileName = file.name;
    parseError = null;
    statusMessage = null;
    columnMapping = null;
    showColumnMapper = false;

    const ext = file.name.split(".").pop()?.toLowerCase();
    rawFileContent = await file.text();

    if (
      ext === "ofx" ||
      ext === "qfx" ||
      rawFileContent.includes("<OFX>") ||
      rawFileContent.includes("<STMTTRN>")
    ) {
      detectedFormat = ext === "qfx" ? "qfx" : "ofx";
      tableData = [];
      tableRows = [];
      if (importMode === "smart") await runSmartIngest();
      return;
    }

    detectedFormat = "csv";
    tableData = parseCSVText(rawFileContent, delimiter);
    columnCount = tableData[0]?.length || 0;
    tableRows = asRows(tableData);

    const hint = (file.name + " " + rawFileContent.slice(0, 2000)).toLowerCase();
    for (const b of [
      { key: "kotak", name: "Kotak" },
      { key: "811", name: "Kotak" },
      { key: "icici", name: "ICICI" },
      { key: "sbi", name: "SBI" },
      { key: "hdfc", name: "HDFC" },
      { key: "chase", name: "Chase" }
    ]) {
      if (hint.includes(b.key)) {
        const foundPreset = presets.find((p) => p.name.toLowerCase().includes(b.name.toLowerCase()));
        if (foundPreset) {
          selectedPresetName = foundPreset.name;
          if (foundPreset.delimiter) delimiter = foundPreset.delimiter;
        }
        const foundTpl = templates.find((t) => t.name.toLowerCase().includes(b.name.toLowerCase()));
        if (foundTpl) {
          selectedTemplateName = foundTpl.name;
          templateDraft = foundTpl.content || "";
          saveTemplateAsName = foundTpl.name;
        }
        break;
      }
    }

    if (importMode === "smart") {
      await runSmartIngest();
    } else if (importMode === "template") {
      await refreshImportPreview();
    }
  }

  function syncMapperDraft(mapping?: ColumnMapping | null) {
    if (mapping) {
      mapperDraft = { ...mapping };
      mapperAmountMode =
        mapping.amount >= 0 && mapping.debit < 0 && mapping.credit < 0 ? "single" : "split";
      return;
    }
    const firstNonEmpty = Math.max(
      0,
      tableData.findIndex((row) => row.some((c) => String(c ?? "").trim() !== ""))
    );
    mapperDraft = {
      header_row: firstNonEmpty,
      date: 0,
      description: columnCount > 1 ? 1 : -1,
      debit: columnCount > 2 ? 2 : -1,
      credit: columnCount > 3 ? 3 : -1,
      amount: -1,
      dr_cr: -1,
      currency: -1
    };
    mapperAmountMode = "split";
  }

  async function runSmartIngest() {
    if (!rawFileContent.trim() && tableData.length === 0) return;
    smartLoading = true;
    parseError = null;
    statusMessage = null;
    try {
      const contentToIngest =
        detectedFormat === "csv" && tableData.length > 0 ? toCSVContent(tableData) : rawFileContent;
      const res = await fetch("/api/import/statement", {
        method: "POST",
        headers: { "Content-Type": "application/json", ...authHeaders() },
        body: JSON.stringify({
          format: detectedFormat,
          content: contentToIngest,
          delimiter,
          base_account: baseAccount,
          column_mapping: columnMapping ?? undefined
        })
      });
      const data = (await res.json()) as {
        error?: string;
        message?: string;
        total_parsed?: number;
        auto_categorized?: number;
        possible_duplicates?: number;
        detected_mapping?: ColumnMapping;
        transactions?: StagedTransaction[];
      };
      if (!res.ok) {
        parseError = data.message || data.error || `Parse failed (${res.status})`;
        if (tableData.length > 0 && detectedFormat === "csv") {
          syncMapperDraft(columnMapping);
          showColumnMapper = true;
        }
        return;
      }

      if (data.detected_mapping) syncMapperDraft(data.detected_mapping);

      stagedTransactions = (data.transactions || []).map((tx) => {
        if (!tx.selected_account && !tx.suggested_account && !tx.is_duplicate) {
          const query = [tx.payee, tx.memo].filter(Boolean).join(" ");
          const matches = findMatch(query, accountTfIdf);
          if (matches.length > 0) {
            const preferred = tx.is_debit
              ? matches.find(([acc, s]) => s > 0.02 && acc.startsWith("Expenses:")) ||
                matches.find(
                  ([acc, s]) => s > 0.03 && (acc.startsWith("Liabilities:") || acc.startsWith("Assets:"))
                )
              : matches.find(([acc, s]) => s > 0.02 && acc.startsWith("Income:")) ||
                matches.find(
                  ([acc, s]) => s > 0.03 && (acc.startsWith("Assets:") || acc.startsWith("Liabilities:"))
                );
            if (preferred) {
              return {
                ...tx,
                suggested_account: preferred[0],
                selected_account: "",
                confidence: "ai_suggested"
              };
            }
          }
        }
        return tx;
      });

      includedStaged = stagedTransactions.map((tx) => !tx.is_duplicate && Boolean(tx.selected_account));
      const suggestedCount = stagedTransactions.filter(
        (t) => !t.selected_account && Boolean(t.suggested_account)
      ).length;
      statusMessage = `Parsed ${data.total_parsed ?? stagedTransactions.length} rows (${data.auto_categorized ?? 0} rule-matched, ${suggestedCount} suggested, ${data.possible_duplicates ?? 0} duplicates).`;
    } finally {
      smartLoading = false;
    }
  }

  async function applyColumnMapping() {
    columnMapping = {
      ...mapperDraft,
      debit: mapperAmountMode === "split" ? mapperDraft.debit : -1,
      credit: mapperAmountMode === "split" ? mapperDraft.credit : -1,
      amount: mapperAmountMode === "single" ? mapperDraft.amount : -1,
      dr_cr: mapperAmountMode === "single" ? mapperDraft.dr_cr : -1
    };
    await runSmartIngest();
    if (!parseError) showColumnMapper = false;
  }

  async function loadRules() {
    rulesLoading = true;
    try {
      const res = await apiGet<{ rules?: ImportRule[] }>("/api/rules").catch(() => ({ rules: [] }));
      rules = res.rules || [];
    } finally {
      rulesLoading = false;
    }
  }

  function openRuleStudio(tx?: StagedTransaction) {
    void loadRules();
    if (tx) {
      const payeeClean = extractShortNote(tx.payee, tx.memo);
      editingRule = {
        name: `Auto: ${payeeClean || tx.payee}`,
        priority: 100,
        enabled: true,
        payee_pattern: payeeClean || tx.payee,
        payee_match_type: "contains",
        memo_pattern: "",
        memo_match_type: "contains",
        tx_type: tx.is_debit ? "debit" : "credit",
        target_account: tx.selected_account || tx.suggested_account || "Expenses:Misc",
        tags: [],
        flag_for_review: false
      };
      rawRuleTags = "";
      ruleTab = "edit";
    } else {
      ruleTab = "list";
    }
    ruleStudioOpen = true;
  }

  async function saveRule() {
    if (!editingRule.name.trim() || !editingRule.target_account.trim()) return;
    editingRule.tags = rawRuleTags
      .split(",")
      .map((t) => t.trim())
      .filter(Boolean);
    const res = await fetch("/api/rules", {
      method: "POST",
      headers: { "Content-Type": "application/json", ...authHeaders() },
      body: JSON.stringify(editingRule)
    });
    if (res.ok) {
      await loadRules();
      ruleTab = "list";
      if (stagedTransactions.length > 0) await runSmartIngest();
    }
  }

  async function deleteRule(id?: number) {
    if (!id) return;
    const res = await fetch(`/api/rules/${id}`, {
      method: "DELETE",
      headers: authHeaders()
    });
    if (res.ok) {
      rules = rules.filter((r) => r.id !== id);
      if (stagedTransactions.length > 0) await runSmartIngest();
    }
  }

  async function quickSaveRule(tx: StagedTransaction) {
    if (!tx.selected_account) return;
    const payeeClean = extractShortNote(tx.payee, tx.memo) || tx.payee;
    const res = await fetch("/api/rules", {
      method: "POST",
      headers: { "Content-Type": "application/json", ...authHeaders() },
      body: JSON.stringify({
        name: `Rule: ${payeeClean}`,
        payee_pattern: payeeClean,
        payee_match_type: "contains",
        tx_type: tx.is_debit ? "debit" : "credit",
        target_account: tx.selected_account,
        priority: 100,
        enabled: true
      })
    });
    if (res.ok) {
      statusMessage = `Saved rule "${payeeClean}" -> ${tx.selected_account}`;
      stagedTransactions = stagedTransactions.map((t) =>
        !t.selected_account && t.payee.toLowerCase().includes(payeeClean.toLowerCase())
          ? { ...t, selected_account: tx.selected_account, confidence: "rule" }
          : t
      );
    }
  }

  function applyBulkAccount(acc: string) {
    if (!acc) return;
    stagedTransactions = stagedTransactions.map((tx, idx) =>
      includedStaged[idx] ? { ...tx, selected_account: acc, confidence: "manual" } : tx
    );
    bulkAccount = "";
  }

  function openReviewLedgerModal() {
    const selected = stagedTransactions.filter((_, i) => includedStaged[i]);
    if (selected.length === 0 || !baseAccount.trim()) return;
    reviewLedgerContent = generateLedgerText(
      selected,
      baseAccount.trim(),
      runtime.defaultCurrency || "INR",
      groupByDate
    );
    reviewLedgerOpen = true;
  }

  async function commitStaged(useRawReview = false) {
    const selected = stagedTransactions.filter((_, i) => includedStaged[i]);
    if (selected.length === 0 && (!useRawReview || !reviewLedgerContent.trim())) return;
    smartCommitting = true;
    try {
      const res = await fetch("/api/import/commit", {
        method: "POST",
        headers: { "Content-Type": "application/json", ...authHeaders() },
        body: JSON.stringify({
          base_account: baseAccount,
          commodity: runtime.defaultCurrency || "INR",
          group_by_date: groupByDate,
          raw_content: useRawReview ? reviewLedgerContent : "",
          transactions: selected
        })
      });
      const data = (await res.json()) as { success?: boolean; committed_count?: number };
      if (res.ok && data.success) {
        statusMessage = `Appended ${data.committed_count ?? selected.length} transactions to ledger.`;
        stagedTransactions = stagedTransactions.filter((_, i) => !includedStaged[i]);
        includedStaged = stagedTransactions.map(() => true);
        reviewLedgerOpen = false;
        void triggerSync({ journal: true });
      } else {
        parseError = "Failed to commit transactions.";
      }
    } finally {
      smartCommitting = false;
    }
  }

  // ---- Classic Template Editor actions ----
  async function refreshImportPreview() {
    if (tableData.length === 0 || !selectedTemplateName) return;
    const res = await fetch("/api/import/preview", {
      method: "POST",
      headers: { "Content-Type": "application/json", ...authHeaders() },
      body: JSON.stringify({
        template: selectedTemplateName,
        content: toCSVContent(tableData),
        delimiter,
        dry_run: true
      })
    });
    const data = (await res.json()) as {
      rows?: ImportPreviewRow[];
      error?: { message?: string };
    };
    if (!res.ok || data.error) {
      parseError = data.error?.message || "Failed to generate parsed preview";
      previewRows = [];
      includedPreviewRows = [];
      return;
    }
    parseError = null;
    previewRows = data.rows || [];
    includedPreviewRows = previewRows.map((r) => Boolean(r.valid));
  }

  async function saveTemplate(name: string) {
    const clean = name.trim();
    if (!clean) return;
    const res = await fetch("/api/templates/upsert", {
      method: "POST",
      headers: { "Content-Type": "application/json", ...authHeaders() },
      body: JSON.stringify({ name: clean, content: templateDraft })
    });
    if (res.ok) {
      const r = await apiGet<{ templates?: ImportTemplate[] }>("/api/templates").catch(() => ({
        templates: []
      }));
      templates = r.templates || [];
      selectedTemplateName = clean;
      saveTemplateAsName = clean;
      templateSaveModalOpen = false;
      statusMessage = `Saved template "${clean}".`;
    }
  }

  async function deleteSelectedTemplate() {
    if (!selectedTemplate || selectedTemplate.template_type === "builtin") return;
    const res = await fetch("/api/templates/delete", {
      method: "POST",
      headers: { "Content-Type": "application/json", ...authHeaders() },
      body: JSON.stringify({ name: selectedTemplate.name })
    });
    if (res.ok) {
      templates = templates.filter((t) => t.name !== selectedTemplate.name);
      selectedTemplateName = templates[0]?.name || "";
      templateDraft = templates[0]?.content || "";
      statusMessage = "Template deleted.";
    }
  }

  async function savePreset() {
    const clean = newPresetName.trim();
    if (!clean) return;
    const columnMappings = Object.fromEntries(
      Array.from({ length: columnCount }, (_, i) => [columnName(i), columnName(i)])
    );
    const res = await fetch("/api/import/presets", {
      method: "POST",
      headers: { "Content-Type": "application/json", ...authHeaders() },
      body: JSON.stringify({
        name: clean,
        column_mappings: columnMappings,
        date_format: "",
        default_accounts: {},
        delimiter
      })
    });
    if (res.ok) {
      newPresetName = "";
      const r = await apiGet<{ presets?: ImportPreset[] }>("/api/import/presets").catch(() => ({
        presets: []
      }));
      presets = r.presets || [];
      selectedPresetName = clean;
      statusMessage = `Saved preset "${clean}".`;
    }
  }

  async function saveRenderedToFile() {
    if (!destinationFile.trim() || !renderedPreview.trim()) return;
    savingToFile = true;
    try {
      const res = await fetch("/api/editor/save", {
        method: "POST",
        headers: { "Content-Type": "application/json", ...authHeaders() },
        body: JSON.stringify({
          name: destinationFile.trim(),
          content: renderedPreview,
          operation: "overwrite"
        })
      });
      const data = (await res.json()) as { saved?: boolean; message?: string };
      if (data.saved) {
        saveToFileOpen = false;
        statusMessage = `Saved rendered entries to ${destinationFile}.`;
        void triggerSync({ journal: true });
      } else {
        parseError = data.message || "Failed to save file.";
      }
    } finally {
      savingToFile = false;
    }
  }
</script>

<div class="space-y-4">
  <!-- Mode Switcher & Actions -->
  <div class="flex flex-wrap items-center justify-between gap-3">
    <div class="inline-flex rounded-lg border border-[var(--border)] bg-[var(--surface)] p-0.5 text-xs">
      <button
        type="button"
        onclick={() => (importMode = "smart")}
        class="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-md cursor-pointer {importMode ===
        'smart'
          ? 'bg-[var(--accent)] text-[var(--accent-fg)] font-semibold'
          : 'text-[var(--muted)] hover:text-[var(--text)]'}"
      >
        <WandSparkles class="w-3.5 h-3.5" />
        <span>Smart Ingestion</span>
      </button>
      <button
        type="button"
        onclick={() => {
          importMode = "template";
          if (tableData.length > 0 && previewRows.length === 0) void refreshImportPreview();
        }}
        class="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-md cursor-pointer {importMode ===
        'template'
          ? 'bg-[var(--accent)] text-[var(--accent-fg)] font-semibold'
          : 'text-[var(--muted)] hover:text-[var(--text)]'}"
      >
        <Code2 class="w-3.5 h-3.5" />
        <span>Template Editor</span>
      </button>
    </div>

    <div class="flex items-center gap-2">
      <button
        type="button"
        onclick={() => openRuleStudio()}
        class="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium border border-[var(--border)] bg-[var(--surface)] hover:border-[var(--accent)] cursor-pointer"
      >
        <SlidersHorizontal class="w-3.5 h-3.5" />
        <span>Rule Studio</span>
      </button>
    </div>
  </div>

  {#if parseError}
    <div class="rounded-xl border border-[var(--loss)]/30 bg-[var(--loss)]/10 px-4 py-2.5 text-xs text-[var(--loss)] flex items-center justify-between gap-2">
      <span>{parseError}</span>
      {#if tableData.length > 0 && detectedFormat === "csv"}
        <button
          type="button"
          onclick={() => {
            syncMapperDraft(columnMapping);
            showColumnMapper = true;
          }}
          class="px-2.5 py-1 rounded bg-[var(--surface)] border border-[var(--border)] text-[var(--text)] cursor-pointer shrink-0"
        >
          Map Columns
        </button>
      {/if}
    </div>
  {/if}

  {#if statusMessage}
    <div class="rounded-xl border border-[var(--gain)]/30 bg-[var(--gain)]/10 px-4 py-2 text-xs text-[var(--gain)]">
      {statusMessage}
    </div>
  {/if}

  {#if importMode === "smart"}
    <div class="grid grid-cols-1 lg:grid-cols-12 gap-4">
      <!-- Left Controls -->
      <div class="lg:col-span-4 space-y-4">
        <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-4 space-y-3 text-xs">
          <div class="font-semibold text-sm">1. Upload Statement</div>
          <label class="flex flex-col items-center justify-center gap-1.5 border border-dashed border-[var(--border)] hover:border-[var(--accent)] rounded-lg p-5 text-center cursor-pointer bg-[var(--surface-elevated)]/40">
            <Upload class="w-5 h-5 text-[var(--muted)]" />
            <span class="font-medium">{uploadedFileName || "Select CSV, TXT, OFX, or QFX file"}</span>
            <span class="text-[11px] text-[var(--muted)]">Auto-detects format and suggests source account</span>
            <input type="file" accept=".csv,.txt,.ofx,.qfx" onchange={handleFileSelect} class="hidden" />
          </label>
          {#if uploadedFileName}
            <div class="flex items-center justify-between text-[11px] text-[var(--muted)]">
              <span>File: <strong class="text-[var(--text)]">{uploadedFileName}</strong></span>
              <span class="px-1.5 py-0.5 rounded bg-[var(--surface-elevated)] uppercase font-mono">{detectedFormat}</span>
            </div>
          {/if}
        </div>

        <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-4 space-y-3 text-xs">
          <div class="font-semibold text-sm">2. Statement Account</div>
          <div>
            <label class="block text-[var(--muted)] mb-1" for="smart-base-acc">Source / Bank Account</label>
            <input
              id="smart-base-acc"
              type="text"
              list="smart-base-accounts"
              bind:value={baseAccount}
              placeholder={topSuggestedBaseAccount
                ? `Suggested: ${topSuggestedBaseAccount}`
                : "e.g. Assets:Checking:ICICI"}
              class="w-full px-3 py-1.5 rounded-lg bg-[var(--surface-elevated)] border border-[var(--border)] outline-none"
            />
            <datalist id="smart-base-accounts">
              {#each orderedBaseAccounts as acc}
                <option value={acc}></option>
              {/each}
            </datalist>
            {#if topSuggestedBaseAccount && !baseAccount}
              <div class="mt-1.5 flex items-center justify-between gap-2 text-[11px] text-[var(--muted)]">
                <span class="truncate">Header match: <strong class="text-[var(--text)]">{topSuggestedBaseAccount}</strong></span>
                <button
                  type="button"
                  onclick={() => (baseAccount = topSuggestedBaseAccount)}
                  class="px-2 py-0.5 rounded bg-[var(--accent)]/15 text-[var(--accent)] font-medium cursor-pointer shrink-0"
                >
                  Select
                </button>
              </div>
            {/if}
          </div>

          <div class="flex items-end gap-2">
            <div class="w-20">
              <label class="block text-[var(--muted)] mb-1" for="smart-delim">Delimiter</label>
              <input
                id="smart-delim"
                type="text"
                maxlength="1"
                bind:value={delimiter}
                onblur={() => {
                  if (rawFileContent && detectedFormat === "csv") {
                    tableData = parseCSVText(rawFileContent, delimiter);
                    columnCount = tableData[0]?.length || 0;
                    tableRows = asRows(tableData);
                    void runSmartIngest();
                  }
                }}
                class="w-full px-2.5 py-1.5 rounded-lg bg-[var(--surface-elevated)] border border-[var(--border)] font-mono outline-none"
              />
            </div>
            {#if tableData.length > 0 && detectedFormat === "csv"}
              <button
                type="button"
                onclick={() => {
                  syncMapperDraft(columnMapping);
                  showColumnMapper = true;
                }}
                class="inline-flex items-center gap-1 px-2.5 py-1.5 rounded-lg border border-[var(--border)] hover:border-[var(--accent)] cursor-pointer"
              >
                <Columns3 class="w-3.5 h-3.5" />
                <span>Map Columns</span>
              </button>
            {/if}
            <button
              type="button"
              disabled={smartLoading || (!rawFileContent && tableData.length === 0)}
              onclick={runSmartIngest}
              class="ml-auto px-3 py-1.5 rounded-lg bg-[var(--surface-elevated)] border border-[var(--border)] hover:border-[var(--accent)] cursor-pointer disabled:opacity-50"
            >
              {smartLoading ? "Parsing…" : "Re-analyze"}
            </button>
          </div>
        </div>

        <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-4 space-y-3 text-xs">
          <div class="font-semibold text-sm">3. Review &amp; Append to Ledger</div>
          <label class="flex items-center gap-2 text-[var(--muted)] cursor-pointer">
            <input type="checkbox" bind:checked={groupByDate} />
            <span>Club multiple debit entries per date into single transaction</span>
          </label>
          {#if !runtime.readonly}
            <div class="grid grid-cols-1 gap-2">
              <button
                type="button"
                disabled={selectedStagedCount === 0 || !baseAccount.trim()}
                onclick={openReviewLedgerModal}
                class="w-full px-3 py-2 rounded-lg border border-[var(--accent)] text-[var(--accent)] font-medium hover:bg-[var(--accent)]/10 disabled:opacity-40 cursor-pointer"
              >
                Review / Edit Ledger ({selectedStagedCount})
              </button>
              <button
                type="button"
                disabled={selectedStagedCount === 0 || !baseAccount.trim() || smartCommitting}
                onclick={() => commitStaged(false)}
                class="w-full px-3 py-2 rounded-lg bg-[var(--accent)] text-[var(--accent-fg)] font-medium hover:opacity-90 disabled:opacity-40 cursor-pointer"
              >
                {smartCommitting ? "Committing…" : `Quick Commit (${selectedStagedCount})`}
              </button>
            </div>
          {/if}
        </div>
      </div>

      <!-- Right Staging Table -->
      <div class="lg:col-span-8">
        {#if stagedTransactions.length === 0}
          <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-10 text-center space-y-2">
            <Sparkles class="w-8 h-8 text-[var(--muted)] mx-auto" />
            <p class="text-sm font-semibold">No Statement Staged</p>
            <p class="text-xs text-[var(--muted)]">
              Upload a bank statement (CSV, TXT, OFX, QFX) on the left to parse, detect duplicates, and auto-categorize transactions.
            </p>
          </div>
        {:else}
          <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] overflow-hidden">
            <div class="px-4 py-3 border-b border-[var(--border)] flex flex-wrap items-center justify-between gap-2 text-xs">
              <div class="flex items-center gap-1.5">
                {#each [
                  { id: "all", label: `All (${stagedTransactions.length})` },
                  { id: "ready", label: `Ready (${stagedTransactions.filter((t) => !t.is_duplicate && t.selected_account).length})` },
                  { id: "unassigned", label: `Unassigned (${stagedTransactions.filter((t) => !t.selected_account).length})` },
                  { id: "duplicate", label: `Duplicates (${stagedTransactions.filter((t) => t.is_duplicate).length})` }
                ] as f}
                  <button
                    type="button"
                    onclick={() => (stageFilter = f.id as typeof stageFilter)}
                    class="px-2.5 py-1 rounded-full border cursor-pointer {stageFilter === f.id
                      ? 'bg-[var(--accent)] text-[var(--accent-fg)] border-[var(--accent)] font-medium'
                      : 'border-[var(--border)] text-[var(--muted)] hover:text-[var(--text)]'}"
                  >
                    {f.label}
                  </button>
                {/each}
              </div>

              <div class="flex items-center gap-1.5">
                <input
                  type="text"
                  list="all-accounts-datalist"
                  bind:value={bulkAccount}
                  placeholder="Bulk assign account…"
                  class="px-2.5 py-1 rounded-md bg-[var(--surface-elevated)] border border-[var(--border)] outline-none w-48"
                />
                <button
                  type="button"
                  onclick={() => applyBulkAccount(bulkAccount)}
                  disabled={!bulkAccount || selectedStagedCount === 0}
                  class="px-2.5 py-1 rounded-md border border-[var(--border)] hover:border-[var(--accent)] disabled:opacity-40 cursor-pointer"
                >
                  Apply
                </button>
              </div>
            </div>

            <datalist id="all-accounts-datalist">
              {#each runtime.accounts as acc}
                <option value={acc}></option>
              {/each}
            </datalist>

            <div class="overflow-x-auto max-h-[600px] overflow-y-auto">
              <table class="w-full text-xs text-left">
                <thead class="uppercase tracking-wider text-[var(--muted)] border-b border-[var(--border)] bg-[var(--surface-elevated)]/60 sticky top-0">
                  <tr>
                    <th class="py-2 px-3">
                      <input
                        type="checkbox"
                        checked={stagedTransactions.length > 0 && includedStaged.every(Boolean)}
                        onchange={(e) => {
                          const checked = (e.currentTarget as HTMLInputElement).checked;
                          includedStaged = stagedTransactions.map(() => checked);
                        }}
                      />
                    </th>
                    <th class="py-2 px-3">Date</th>
                    <th class="py-2 px-3">Payee / Memo</th>
                    <th class="py-2 px-3">Category Account</th>
                    <th class="py-2 px-3 text-right">Amount</th>
                    <th class="py-2 px-3 text-right">Rule</th>
                  </tr>
                </thead>
                <tbody class="divide-y divide-[var(--border)] tabular-nums">
                  {#each filteredStagedIndices as idx}
                    {@const tx = stagedTransactions[idx]}
                    <tr class="hover:bg-[var(--surface-elevated)]/40 {tx.is_duplicate ? 'opacity-65' : ''}">
                      <td class="py-2 px-3">
                        <input type="checkbox" bind:checked={includedStaged[idx]} />
                      </td>
                      <td class="py-2 px-3 text-[var(--muted)] whitespace-nowrap">{tx.date}</td>
                      <td class="py-2 px-3 min-w-[200px]">
                        <div class="font-medium flex items-center gap-1.5">
                          <span>{tx.payee}</span>
                          {#if tx.is_duplicate}
                            <span class="px-1.5 py-0.5 rounded bg-[var(--warning)]/15 text-[var(--warning)] text-[10px] font-semibold">
                              duplicate
                            </span>
                          {/if}
                          {#if tx.matched_rule_name}
                            <span class="px-1.5 py-0.5 rounded bg-[var(--gain)]/15 text-[var(--gain)] text-[10px]">
                              {tx.matched_rule_name}
                            </span>
                          {/if}
                        </div>
                        {#if tx.memo}
                          <div class="text-[11px] text-[var(--muted)] truncate max-w-xs">{tx.memo}</div>
                        {/if}
                      </td>
                      <td class="py-2 px-3 min-w-[220px]">
                        <div class="flex items-center gap-1">
                          <input
                            type="text"
                            list="all-accounts-datalist"
                            bind:value={tx.selected_account}
                            placeholder={tx.suggested_account || "Select account…"}
                            class="w-full px-2 py-1 rounded bg-[var(--surface-elevated)] border border-[var(--border)] outline-none"
                          />
                          {#if !tx.selected_account && tx.suggested_account}
                            <button
                              type="button"
                              title={`Accept suggestion: ${tx.suggested_account}`}
                              onclick={() => {
                                tx.selected_account = tx.suggested_account;
                                includedStaged[idx] = !tx.is_duplicate;
                              }}
                              class="p-1 rounded bg-[var(--accent)]/15 text-[var(--accent)] hover:bg-[var(--accent)]/25 cursor-pointer shrink-0"
                            >
                              <Check class="w-3.5 h-3.5" />
                            </button>
                          {/if}
                        </div>
                      </td>
                      <td class="py-2 px-3 text-right font-semibold whitespace-nowrap {tx.is_debit ? 'text-[var(--loss)]' : 'text-[var(--gain)]'}">
                        {tx.is_debit ? "-" : "+"}{formatMoney(tx.amount || 0)}
                      </td>
                      <td class="py-2 px-3 text-right whitespace-nowrap">
                        <div class="inline-flex items-center gap-1">
                          {#if tx.selected_account && !runtime.readonly}
                            <button
                              type="button"
                              title="Quick save rule"
                              onclick={() => quickSaveRule(tx)}
                              class="p-1 rounded border border-[var(--border)] hover:border-[var(--accent)] cursor-pointer"
                            >
                              <Save class="w-3 h-3" />
                            </button>
                          {/if}
                          <button
                            type="button"
                            title="Create rule in Rule Studio"
                            onclick={() => openRuleStudio(tx)}
                            class="p-1 rounded border border-[var(--border)] hover:border-[var(--accent)] cursor-pointer"
                          >
                            <SlidersHorizontal class="w-3 h-3" />
                          </button>
                        </div>
                      </td>
                    </tr>
                  {/each}
                </tbody>
              </table>
            </div>
          </div>
        {/if}
      </div>
    </div>
  {:else if importMode === "template"}
    <!-- Classic Template Editor (Handlebars / Freemarker-style + Parsed Data + Rendered Postings) -->
    <div class="grid grid-cols-1 lg:grid-cols-12 gap-4">
      <!-- Left: Template Editor + Rendered Postings Preview -->
      <div class="lg:col-span-5 space-y-4">
        <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-4 space-y-3 text-xs">
          <div class="flex flex-wrap items-center gap-2">
            <select
              bind:value={selectedTemplateName}
              onchange={() => {
                const tpl = templates.find((t) => t.name === selectedTemplateName);
                templateDraft = tpl?.content || "";
                saveTemplateAsName = tpl?.name || "";
                if (tableData.length > 0) void refreshImportPreview();
              }}
              class="flex-1 min-w-[160px] px-2.5 py-1.5 rounded-lg bg-[var(--surface-elevated)] border border-[var(--border)] outline-none"
            >
              {#each templates as tpl}
                <option value={tpl.name}>
                  {tpl.name} ({tpl.template_type || "custom"})
                </option>
              {/each}
            </select>
            {#if !runtime.readonly}
              <button
                type="button"
                onclick={() => {
                  saveTemplateAsName = "";
                  templateSaveModalOpen = true;
                }}
                title="Create template"
                class="p-1.5 rounded-lg border border-[var(--border)] hover:border-[var(--accent)] cursor-pointer"
              >
                <FilePlus2 class="w-3.5 h-3.5" />
              </button>
              <button
                type="button"
                disabled={selectedTemplate?.template_type === "builtin"}
                onclick={() => saveTemplate(selectedTemplateName)}
                title={selectedTemplate?.template_type === "builtin"
                  ? "Builtin templates cannot be overwritten"
                  : "Save template"}
                class="p-1.5 rounded-lg border border-[var(--border)] hover:border-[var(--accent)] disabled:opacity-40 cursor-pointer"
              >
                <Save class="w-3.5 h-3.5" />
              </button>
              <button
                type="button"
                disabled={selectedTemplate?.template_type === "builtin"}
                onclick={deleteSelectedTemplate}
                title="Delete custom template"
                class="p-1.5 rounded-lg border border-[var(--border)] hover:border-[var(--loss)] text-[var(--muted)] hover:text-[var(--loss)] disabled:opacity-40 cursor-pointer"
              >
                <Trash2 class="w-3.5 h-3.5" />
              </button>
            {/if}
          </div>

          <div>
            <div class="text-[11px] font-semibold uppercase tracking-wider text-[var(--muted)] mb-1">
              Template Editor (Handlebars)
            </div>
            <textarea
              bind:value={templateDraft}
              readonly={selectedTemplate?.template_type === "builtin" && runtime.readonly}
              spellcheck="false"
              rows="11"
              class="w-full p-3 rounded-lg bg-[var(--surface-elevated)] border border-[var(--border)] font-mono text-[11px] leading-relaxed outline-none"
            ></textarea>
          </div>
        </div>

        <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-4 space-y-2.5 text-xs">
          <div class="flex items-center justify-between gap-2">
            <span class="text-[11px] font-semibold uppercase tracking-wider text-[var(--muted)]">
              Rendered Postings Preview
            </span>
            <div class="flex items-center gap-1.5">
              <button
                type="button"
                disabled={!renderedPreview}
                onclick={() => navigator.clipboard.writeText(renderedPreview)}
                class="inline-flex items-center gap-1 px-2 py-1 rounded border border-[var(--border)] hover:border-[var(--accent)] disabled:opacity-40 cursor-pointer"
              >
                <Copy class="w-3 h-3" />
                <span>Copy</span>
              </button>
              {#if !runtime.readonly}
                <button
                  type="button"
                  disabled={!renderedPreview || selectedPreviewCount === 0}
                  onclick={() => (saveToFileOpen = true)}
                  class="inline-flex items-center gap-1 px-2.5 py-1 rounded bg-[var(--accent)] text-[var(--accent-fg)] font-medium disabled:opacity-40 cursor-pointer"
                >
                  <Save class="w-3 h-3" />
                  <span>Save to File</span>
                </button>
              {/if}
            </div>
          </div>
          {#if previewRenderError}
            <div class="p-2 rounded bg-[var(--loss)]/10 text-[var(--loss)]">{previewRenderError}</div>
          {/if}
          <pre class="p-3 rounded-lg bg-[var(--surface-elevated)] border border-[var(--border)] font-mono text-[11px] leading-relaxed overflow-auto max-h-[340px] whitespace-pre-wrap">{renderedPreview || "Upload a CSV file and select a template to preview generated postings."}</pre>
        </div>
      </div>

      <!-- Right: Upload, Presets, Parsed Data Table -->
      <div class="lg:col-span-7 space-y-4">
        <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-4 space-y-3 text-xs">
          <div class="flex flex-wrap items-end gap-3">
            <div class="flex-1 min-w-[200px]">
              <label class="block text-[var(--muted)] mb-1" for="tpl-file">Statement CSV / TXT</label>
              <input id="tpl-file" type="file" accept=".csv,.txt" onchange={handleFileSelect} class="text-xs" />
            </div>
            <div class="w-44">
              <label class="block text-[var(--muted)] mb-1" for="tpl-preset">Preset</label>
              <select
                id="tpl-preset"
                bind:value={selectedPresetName}
                onchange={() => {
                  if (selectedPreset?.delimiter) delimiter = selectedPreset.delimiter;
                  if (rawFileContent) {
                    tableData = parseCSVText(rawFileContent, delimiter);
                    columnCount = tableData[0]?.length || 0;
                    tableRows = asRows(tableData);
                    void refreshImportPreview();
                  }
                }}
                class="w-full px-2.5 py-1.5 rounded-lg bg-[var(--surface-elevated)] border border-[var(--border)]"
              >
                {#each presets as p}
                  <option value={p.name}>{p.name}</option>
                {/each}
              </select>
            </div>
            <div class="w-20">
              <label class="block text-[var(--muted)] mb-1" for="tpl-delim">Delimiter</label>
              <input
                id="tpl-delim"
                type="text"
                maxlength="1"
                bind:value={delimiter}
                onblur={() => {
                  if (rawFileContent) {
                    tableData = parseCSVText(rawFileContent, delimiter);
                    columnCount = tableData[0]?.length || 0;
                    tableRows = asRows(tableData);
                    void refreshImportPreview();
                  }
                }}
                class="w-full px-2.5 py-1.5 rounded-lg bg-[var(--surface-elevated)] border border-[var(--border)] font-mono"
              />
            </div>
          </div>

          <div class="flex flex-wrap items-center justify-between gap-2 pt-2 border-t border-[var(--border)]/60">
            <div class="flex items-center gap-4">
              <label class="inline-flex items-center gap-1.5 cursor-pointer">
                <input type="checkbox" bind:checked={reverseRows} />
                <span>Reverse</span>
              </label>
              <label class="inline-flex items-center gap-1.5 cursor-pointer">
                <input type="checkbox" bind:checked={trimRows} />
                <span>Trim</span>
              </label>
            </div>
            {#if !runtime.readonly}
              <div class="flex items-center gap-1.5">
                <input
                  type="text"
                  bind:value={newPresetName}
                  placeholder="Save preset as…"
                  class="px-2 py-1 rounded bg-[var(--surface-elevated)] border border-[var(--border)] outline-none"
                />
                <button
                  type="button"
                  onclick={savePreset}
                  disabled={!newPresetName.trim()}
                  class="px-2.5 py-1 rounded border border-[var(--border)] hover:border-[var(--accent)] disabled:opacity-40 cursor-pointer"
                >
                  Save Preset
                </button>
              </div>
            {/if}
          </div>
        </div>

        <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] overflow-hidden">
          <div class="px-4 py-2.5 border-b border-[var(--border)] flex items-center justify-between text-xs">
            <span class="font-semibold uppercase tracking-wider text-[var(--muted)]">
              Parsed Data ({previewRows.length} rows · {selectedPreviewCount} selected)
            </span>
            <button
              type="button"
              disabled={tableData.length === 0}
              onclick={refreshImportPreview}
              class="px-2.5 py-1 rounded border border-[var(--border)] hover:border-[var(--accent)] disabled:opacity-40 cursor-pointer"
            >
              Validate Rows
            </button>
          </div>
          {#if previewRows.length === 0}
            <div class="p-8 text-center text-xs text-[var(--muted)]">
              Upload a CSV/TXT file to inspect parsed rows and validation status.
            </div>
          {:else}
            <div class="overflow-x-auto max-h-[540px] overflow-y-auto">
              <table class="w-full text-xs text-left">
                <thead class="uppercase tracking-wider text-[var(--muted)] border-b border-[var(--border)] bg-[var(--surface-elevated)]/60 sticky top-0">
                  <tr>
                    <th class="py-2 px-3">
                      <input
                        type="checkbox"
                        checked={previewRows.length > 0 && includedPreviewRows.every(Boolean)}
                        onchange={(e) => {
                          const checked = (e.currentTarget as HTMLInputElement).checked;
                          includedPreviewRows = previewRows.map(() => checked);
                        }}
                      />
                    </th>
                    <th class="py-2 px-3">Status</th>
                    {#each previewColumns as col}
                      <th class="py-2 px-3">{col}</th>
                    {/each}
                  </tr>
                </thead>
                <tbody class="divide-y divide-[var(--border)]">
                  {#each previewRows as row, idx}
                    <tr class="hover:bg-[var(--surface-elevated)]/40">
                      <td class="py-1.5 px-3 whitespace-nowrap">
                        <label class="inline-flex items-center gap-1.5">
                          <input type="checkbox" bind:checked={includedPreviewRows[idx]} />
                          <span class="font-mono text-[var(--muted)]">{row.index}</span>
                        </label>
                      </td>
                      <td class="py-1.5 px-3">
                        {#if row.valid}
                          <span class="px-1.5 py-0.5 rounded bg-[var(--gain)]/15 text-[var(--gain)] text-[10px] font-semibold">valid</span>
                        {:else}
                          <span class="px-1.5 py-0.5 rounded bg-[var(--loss)]/15 text-[var(--loss)] text-[10px] font-semibold">invalid</span>
                          {#if row.errors?.length}
                            <div class="text-[10px] text-[var(--loss)] mt-0.5">{row.errors.join("; ")}</div>
                          {/if}
                        {/if}
                      </td>
                      {#each previewColumns as col}
                        <td class="py-1.5 px-3 max-w-[180px] truncate">{row.row?.[col] || ""}</td>
                      {/each}
                    </tr>
                  {/each}
                </tbody>
              </table>
            </div>
          {/if}
        </div>
      </div>
    </div>
  {/if}
</div>

<!-- Column Mapper Modal -->
{#if showColumnMapper}
  <div class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50">
    <div class="w-full max-w-3xl rounded-xl bg-[var(--surface)] border border-[var(--border)] shadow-xl overflow-hidden text-xs">
      <div class="px-4 py-3 border-b border-[var(--border)] flex items-center justify-between">
        <span class="text-sm font-semibold">Map Statement Columns</span>
        <button type="button" onclick={() => (showColumnMapper = false)} class="text-[var(--muted)] hover:text-[var(--text)] cursor-pointer">✕</button>
      </div>
      <div class="p-4 space-y-4 max-h-[75vh] overflow-y-auto">
        <div>
          <div class="font-semibold mb-1">1. Header Row (click a row to select)</div>
          <div class="border border-[var(--border)] rounded-lg overflow-auto max-h-44">
            <table class="w-full text-[11px] text-left">
              <tbody>
                {#each tableData.slice(0, 12) as row, idx}
                  <tr
                    onclick={() => (mapperDraft.header_row = idx)}
                    class="cursor-pointer border-b border-[var(--border)]/50 {mapperDraft.header_row === idx
                      ? 'bg-[var(--accent)]/15 font-semibold'
                      : 'hover:bg-[var(--surface-elevated)]'}"
                  >
                    <td class="py-1 px-2 whitespace-nowrap text-[var(--muted)]">Row {idx + 1}</td>
                    {#each row.slice(0, 8) as cell}
                      <td class="py-1 px-2 max-w-[130px] truncate">{cell}</td>
                    {/each}
                  </tr>
                {/each}
              </tbody>
            </table>
          </div>
        </div>

        <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
          <div>
            <label class="block text-[var(--muted)] mb-1" for="cm-date">Date Column (required)</label>
            <select id="cm-date" bind:value={mapperDraft.date} class="w-full px-2.5 py-1.5 rounded bg-[var(--surface-elevated)] border border-[var(--border)]">
              {#each mapperHeaderColumns as col}
                <option value={col.index}>{col.label}</option>
              {/each}
            </select>
          </div>
          <div>
            <label class="block text-[var(--muted)] mb-1" for="cm-desc">Description / Payee Column</label>
            <select id="cm-desc" bind:value={mapperDraft.description} class="w-full px-2.5 py-1.5 rounded bg-[var(--surface-elevated)] border border-[var(--border)]">
              <option value={-1}>— None —</option>
              {#each mapperHeaderColumns as col}
                <option value={col.index}>{col.label}</option>
              {/each}
            </select>
          </div>
        </div>

        <div class="flex items-center gap-4">
          <span class="font-semibold">Amount Format:</span>
          <label class="inline-flex items-center gap-1.5 cursor-pointer">
            <input type="radio" value="split" bind:group={mapperAmountMode} />
            <span>Separate Debit &amp; Credit</span>
          </label>
          <label class="inline-flex items-center gap-1.5 cursor-pointer">
            <input type="radio" value="single" bind:group={mapperAmountMode} />
            <span>Single Amount (+ optional Dr/Cr)</span>
          </label>
        </div>

        <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
          {#if mapperAmountMode === "split"}
            <div>
              <label class="block text-[var(--muted)] mb-1" for="cm-deb">Debit / Outflow</label>
              <select id="cm-deb" bind:value={mapperDraft.debit} class="w-full px-2.5 py-1.5 rounded bg-[var(--surface-elevated)] border border-[var(--border)]">
                <option value={-1}>— None —</option>
                {#each mapperHeaderColumns as col}
                  <option value={col.index}>{col.label}</option>
                {/each}
              </select>
            </div>
            <div>
              <label class="block text-[var(--muted)] mb-1" for="cm-cred">Credit / Inflow</label>
              <select id="cm-cred" bind:value={mapperDraft.credit} class="w-full px-2.5 py-1.5 rounded bg-[var(--surface-elevated)] border border-[var(--border)]">
                <option value={-1}>— None —</option>
                {#each mapperHeaderColumns as col}
                  <option value={col.index}>{col.label}</option>
                {/each}
              </select>
            </div>
          {:else}
            <div>
              <label class="block text-[var(--muted)] mb-1" for="cm-amt">Amount Column</label>
              <select id="cm-amt" bind:value={mapperDraft.amount} class="w-full px-2.5 py-1.5 rounded bg-[var(--surface-elevated)] border border-[var(--border)]">
                <option value={-1}>— None —</option>
                {#each mapperHeaderColumns as col}
                  <option value={col.index}>{col.label}</option>
                {/each}
              </select>
            </div>
            <div>
              <label class="block text-[var(--muted)] mb-1" for="cm-drcr">Dr / Cr Indicator</label>
              <select id="cm-drcr" bind:value={mapperDraft.dr_cr} class="w-full px-2.5 py-1.5 rounded bg-[var(--surface-elevated)] border border-[var(--border)]">
                <option value={-1}>— None —</option>
                {#each mapperHeaderColumns as col}
                  <option value={col.index}>{col.label}</option>
                {/each}
              </select>
            </div>
          {/if}
        </div>
      </div>
      <div class="px-4 py-3 border-t border-[var(--border)] flex items-center justify-between">
        <button
          type="button"
          onclick={async () => {
            columnMapping = null;
            await runSmartIngest();
            showColumnMapper = false;
          }}
          class="px-3 py-1.5 rounded-lg border border-[var(--border)] cursor-pointer"
        >
          Reset to Auto-Detect
        </button>
        <div class="flex items-center gap-2">
          <button type="button" onclick={() => (showColumnMapper = false)} class="px-3 py-1.5 rounded-lg border border-[var(--border)] cursor-pointer">
            Cancel
          </button>
          <button type="button" onclick={applyColumnMapping} class="px-3 py-1.5 rounded-lg bg-[var(--accent)] text-[var(--accent-fg)] font-medium cursor-pointer">
            Apply &amp; Parse
          </button>
        </div>
      </div>
    </div>
  </div>
{/if}

<!-- Rule Studio Modal -->
{#if ruleStudioOpen}
  <div class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50">
    <div class="w-full max-w-2xl rounded-xl bg-[var(--surface)] border border-[var(--border)] shadow-xl overflow-hidden text-xs">
      <div class="px-4 py-3 border-b border-[var(--border)] flex items-center justify-between">
        <span class="text-sm font-semibold">Rule Studio</span>
        <div class="flex items-center gap-2">
          <button
            type="button"
            onclick={() => (ruleTab = "list")}
            class="px-2.5 py-1 rounded {ruleTab === 'list' ? 'bg-[var(--accent)] text-[var(--accent-fg)]' : 'border border-[var(--border)]'} cursor-pointer"
          >
            Rules ({rules.length})
          </button>
          <button
            type="button"
            onclick={() => {
              editingRule = {
                name: "",
                priority: 100,
                enabled: true,
                payee_pattern: "",
                payee_match_type: "contains",
                memo_pattern: "",
                memo_match_type: "contains",
                tx_type: "any",
                target_account: "Expenses:Misc",
                tags: [],
                flag_for_review: false
              };
              rawRuleTags = "";
              ruleTab = "edit";
            }}
            class="px-2.5 py-1 rounded {ruleTab === 'edit' ? 'bg-[var(--accent)] text-[var(--accent-fg)]' : 'border border-[var(--border)]'} cursor-pointer"
          >
            New Rule
          </button>
          <button type="button" onclick={() => (ruleStudioOpen = false)} class="ml-2 text-[var(--muted)] hover:text-[var(--text)] cursor-pointer">✕</button>
        </div>
      </div>

      <div class="p-4 max-h-[70vh] overflow-y-auto">
        {#if ruleTab === "list"}
          {#if rulesLoading}
            <div class="py-6 text-center text-[var(--muted)]">Loading rules…</div>
          {:else if rules.length === 0}
            <div class="py-6 text-center text-[var(--muted)]">No auto-categorization rules yet.</div>
          {:else}
            <div class="divide-y divide-[var(--border)]">
              {#each rules as r}
                <div class="py-2.5 flex items-center justify-between gap-3">
                  <div class="min-w-0">
                    <div class="font-semibold truncate">{r.name}</div>
                    <div class="text-[11px] text-[var(--muted)] truncate">
                      Payee {r.payee_match_type}: "{r.payee_pattern}" → <span class="font-mono">{r.target_account}</span>
                    </div>
                  </div>
                  <div class="flex items-center gap-1.5 shrink-0">
                    <button
                      type="button"
                      onclick={() => {
                        editingRule = { ...r };
                        rawRuleTags = (r.tags || []).join(", ");
                        ruleTab = "edit";
                      }}
                      class="px-2 py-1 rounded border border-[var(--border)] cursor-pointer"
                    >
                      Edit
                    </button>
                    <button
                      type="button"
                      onclick={() => deleteRule(r.id)}
                      class="px-2 py-1 rounded border border-[var(--border)] text-[var(--loss)] cursor-pointer"
                    >
                      Delete
                    </button>
                  </div>
                </div>
              {/each}
            </div>
          {/if}
        {:else}
          <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
            <div>
              <label class="block text-[var(--muted)] mb-1" for="rule-name">Rule Name</label>
              <input id="rule-name" type="text" bind:value={editingRule.name} class="w-full px-2.5 py-1.5 rounded bg-[var(--surface-elevated)] border border-[var(--border)]" />
            </div>
            <div>
              <label class="block text-[var(--muted)] mb-1" for="rule-acc">Target Account</label>
              <input id="rule-acc" type="text" list="all-accounts-datalist" bind:value={editingRule.target_account} class="w-full px-2.5 py-1.5 rounded bg-[var(--surface-elevated)] border border-[var(--border)]" />
            </div>
            <div>
              <label class="block text-[var(--muted)] mb-1" for="rule-payee">Payee Pattern</label>
              <input id="rule-payee" type="text" bind:value={editingRule.payee_pattern} class="w-full px-2.5 py-1.5 rounded bg-[var(--surface-elevated)] border border-[var(--border)]" />
            </div>
            <div>
              <label class="block text-[var(--muted)] mb-1" for="rule-match">Match Type</label>
              <select id="rule-match" bind:value={editingRule.payee_match_type} class="w-full px-2.5 py-1.5 rounded bg-[var(--surface-elevated)] border border-[var(--border)]">
                <option value="contains">Contains</option>
                <option value="exact">Exact</option>
                <option value="regex">Regex</option>
              </select>
            </div>
            <div>
              <label class="block text-[var(--muted)] mb-1" for="rule-txtype">Transaction Type</label>
              <select id="rule-txtype" bind:value={editingRule.tx_type} class="w-full px-2.5 py-1.5 rounded bg-[var(--surface-elevated)] border border-[var(--border)]">
                <option value="any">Any</option>
                <option value="debit">Debit (Outflow)</option>
                <option value="credit">Credit (Inflow)</option>
              </select>
            </div>
            <div>
              <label class="block text-[var(--muted)] mb-1" for="rule-tags">Tags (comma separated)</label>
              <input id="rule-tags" type="text" bind:value={rawRuleTags} class="w-full px-2.5 py-1.5 rounded bg-[var(--surface-elevated)] border border-[var(--border)]" />
            </div>
          </div>
          <div class="mt-4 flex justify-end gap-2">
            <button type="button" onclick={() => (ruleTab = "list")} class="px-3 py-1.5 rounded-lg border border-[var(--border)] cursor-pointer">Cancel</button>
            <button type="button" onclick={saveRule} class="px-3 py-1.5 rounded-lg bg-[var(--accent)] text-[var(--accent-fg)] font-medium cursor-pointer">Save Rule</button>
          </div>
        {/if}
      </div>
    </div>
  </div>
{/if}

<!-- Review Generated Ledger Modal -->
{#if reviewLedgerOpen}
  <div class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50">
    <div class="w-full max-w-3xl rounded-xl bg-[var(--surface)] border border-[var(--border)] shadow-xl overflow-hidden text-xs">
      <div class="px-4 py-3 border-b border-[var(--border)] flex items-center justify-between">
        <span class="text-sm font-semibold">Review &amp; Edit Generated Ledger Entries</span>
        <button type="button" onclick={() => (reviewLedgerOpen = false)} class="text-[var(--muted)] hover:text-[var(--text)] cursor-pointer">✕</button>
      </div>
      <div class="p-4 space-y-3">
        <div class="flex items-center justify-between">
          <span class="text-[var(--muted)]">Edit generated postings before appending to your journal:</span>
          <label class="inline-flex items-center gap-1.5 cursor-pointer">
            <input
              type="checkbox"
              bind:checked={groupByDate}
              onchange={() => {
                const selected = stagedTransactions.filter((_, i) => includedStaged[i]);
                reviewLedgerContent = generateLedgerText(
                  selected,
                  baseAccount.trim(),
                  runtime.defaultCurrency || "INR",
                  groupByDate
                );
              }}
            />
            <span>Club debits by date</span>
          </label>
        </div>
        <textarea
          bind:value={reviewLedgerContent}
          rows="16"
          spellcheck="false"
          class="w-full p-3 rounded-lg bg-[var(--surface-elevated)] border border-[var(--border)] font-mono text-[11px] leading-relaxed outline-none"
        ></textarea>
      </div>
      <div class="px-4 py-3 border-t border-[var(--border)] flex justify-end gap-2">
        <button type="button" onclick={() => (reviewLedgerOpen = false)} class="px-3 py-1.5 rounded-lg border border-[var(--border)] cursor-pointer">Cancel</button>
        <button
          type="button"
          disabled={smartCommitting || !reviewLedgerContent.trim()}
          onclick={() => commitStaged(true)}
          class="px-3 py-1.5 rounded-lg bg-[var(--accent)] text-[var(--accent-fg)] font-medium cursor-pointer disabled:opacity-50"
        >
          {smartCommitting ? "Committing…" : "Confirm & Append to Ledger"}
        </button>
      </div>
    </div>
  </div>
{/if}

<!-- Create Template Modal -->
{#if templateSaveModalOpen}
  <div class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50">
    <div class="w-full max-w-md rounded-xl bg-[var(--surface)] border border-[var(--border)] shadow-xl p-4 space-y-3 text-xs">
      <div class="text-sm font-semibold">Save Template As</div>
      <input
        type="text"
        bind:value={saveTemplateAsName}
        placeholder="Template name…"
        class="w-full px-3 py-1.5 rounded-lg bg-[var(--surface-elevated)] border border-[var(--border)] outline-none"
      />
      <div class="flex justify-end gap-2">
        <button type="button" onclick={() => (templateSaveModalOpen = false)} class="px-3 py-1.5 rounded-lg border border-[var(--border)] cursor-pointer">Cancel</button>
        <button
          type="button"
          disabled={!saveTemplateAsName.trim()}
          onclick={() => saveTemplate(saveTemplateAsName)}
          class="px-3 py-1.5 rounded-lg bg-[var(--accent)] text-[var(--accent-fg)] font-medium cursor-pointer"
        >
          Save
        </button>
      </div>
    </div>
  </div>
{/if}

<!-- Save Rendered Postings to File Modal -->
{#if saveToFileOpen}
  <div class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50">
    <div class="w-full max-w-md rounded-xl bg-[var(--surface)] border border-[var(--border)] shadow-xl p-4 space-y-3 text-xs">
      <div class="text-sm font-semibold">Save Rendered Postings to Ledger File</div>
      <select
        bind:value={destinationFile}
        class="w-full px-3 py-1.5 rounded-lg bg-[var(--surface-elevated)] border border-[var(--border)]"
      >
        {#each ledgerFiles as lf}
          <option value={lf}>{lf}</option>
        {/each}
      </select>
      <input
        type="text"
        bind:value={destinationFile}
        placeholder="Or enter relative ledger file path…"
        class="w-full px-3 py-1.5 rounded-lg bg-[var(--surface-elevated)] border border-[var(--border)] outline-none font-mono"
      />
      <div class="flex justify-end gap-2">
        <button type="button" onclick={() => (saveToFileOpen = false)} class="px-3 py-1.5 rounded-lg border border-[var(--border)] cursor-pointer">Cancel</button>
        <button
          type="button"
          disabled={savingToFile || !destinationFile.trim()}
          onclick={saveRenderedToFile}
          class="px-3 py-1.5 rounded-lg bg-[var(--accent)] text-[var(--accent-fg)] font-medium cursor-pointer"
        >
          {savingToFile ? "Saving…" : "Save"}
        </button>
      </div>
    </div>
  </div>
{/if}
