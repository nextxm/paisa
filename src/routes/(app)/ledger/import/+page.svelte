<script lang="ts">
  import Select from "svelte-select";
  import {
    createEditor as createTemplateEditor,
    editorState as templateEditorState
  } from "$lib/template_editor";
  import {
    createEditor as createPreviewEditor,
    updateContent as updatePreviewContent
  } from "$lib/editor";
  import Dropzone from "$lib/components/Dropzone.svelte";
  import ImportPreviewTable from "$lib/components/ImportPreviewTable.svelte";
  import PresetSelector from "$lib/components/PresetSelector.svelte";
  import RuleStudioModal from "$lib/components/RuleStudioModal.svelte";
  import TransactionStagingTable from "$lib/components/TransactionStagingTable.svelte";
  import { parse, asRows, render as renderJournal } from "$lib/spreadsheet";
  import { findMatch } from "$lib/template_helpers";
  import _ from "lodash";
  import type { EditorView } from "codemirror";
  import { onMount } from "svelte";
  import {
    ajax,
    type ImportPreset,
    type ImportPreviewRow,
    type ImportTemplate,
    type StagedTransaction,
    type IngestionResult
  } from "$lib/utils";
  import { cleanPayee } from "$lib/clean_payee";
  import { fetchConfig } from "$lib/config_client";
  import {
    defaultIncludedFromValidation,
    filterSelectedRows,
    toCSVContent
  } from "$lib/import_preview_utils";
  import { accountTfIdf } from "../../../../store";
  import * as toast from "bulma-toast";
  import FileModal from "$lib/components/FileModal.svelte";
  import Modal from "$lib/components/Modal.svelte";
  import type { PageData } from "./$types";

  let { data: pageData }: { data: PageData } = $props();

  let mode: "smart" | "template" = $state("smart");
  let templates: ImportTemplate[] = $state([]);
  let selectedTemplate: ImportTemplate | null = $state(null);
  let saveAsName: string = $state("");
  let preview = $state("");
  let parseErrorMessage: string | null = $state(null);
  let columnCount: number = $state(0);
  let data: any[][] = $state([]);
  let rows: Array<Record<string, any>> = $state([]);
  let previewRows: ImportPreviewRow[] = $state([]);
  let includedPreviewRows: boolean[] = $state([]);
  let importPresets: ImportPreset[] = $state([]);
  let selectedPreset: ImportPreset | null = $state(null);
  let delimiter: string = $state(",");
  let options: { reverse: boolean; trim: boolean } = $state({ reverse: false, trim: true });
  let importSaving = $state(false);

  // Smart Ingestion State
  let stagedTransactions: StagedTransaction[] = $state([]);
  let includedStaged: boolean[] = $state([]);
  let baseAccount: string = $state("");
  let defaultCurrency: string = $state((globalThis as any)?.USER_CONFIG?.default_currency || "INR");
  let groupByDate: boolean = $state(true);
  let accountsList: string[] = $state([]);
  let ruleStudioOpen: boolean = $state(false);
  let initialRuleForStudio: any = $state(null);
  let reviewLedgerOpen: boolean = $state(false);
  let reviewLedgerContent: string = $state("");
  let rawFileContent: string = $state("");
  let detectedFormat: "csv" | "ofx" | "qfx" | "excel" | "pdf" = $state("csv");

  let templateEditorDom: Element | undefined = $state();
  let templateEditor: EditorView | undefined = $state();

  let previewEditorDom: Element | undefined = $state();
  let previewEditor: EditorView | undefined = $state();

  $effect(() => {
    templates = Array.isArray(pageData.templates) ? pageData.templates : [];
    importPresets = Array.isArray(pageData.importPresets) ? pageData.importPresets : [];
  });

  $effect(() => {
    if (!selectedTemplate && templates.length > 0) {
      selectedTemplate = templates[0];
      saveAsName = selectedTemplate?.name || "";
    }
  });

  $effect(() => {
    if (!selectedPreset && importPresets.length > 0) {
      selectedPreset = _.find(importPresets, { name: "Generic Bank CSV" }) || importPresets[0];
      delimiter = selectedPreset?.delimiter || ",";
    }
  });

  onMount(async () => {
    accountTfIdf.set(pageData.accountTfIdf);
    if (templateEditorDom) {
      templateEditor = createTemplateEditor(selectedTemplate?.content || "", templateEditorDom);
    }
    if (previewEditorDom) {
      previewEditor = createPreviewEditor("", preview, previewEditorDom, { readonly: true });
    }
    try {
      const cfg = await fetchConfig({ background: true });
      if (cfg?.config?.default_currency) {
        defaultCurrency = cfg.config.default_currency;
      }
      if (cfg?.accounts && Array.isArray(cfg.accounts)) {
        accountsList = cfg.accounts;
      }
    } catch {
      // ignore if offline or background fetch fails
    }
  });

  // Header content extracted from raw text or parsed sheet rows
  let statementHeaderContext = $derived.by(() => {
    const parts: string[] = [];
    if (rawFileContent) {
      // First 25 lines of raw statement text
      const lines = rawFileContent.split(/\r?\n/).slice(0, 25);
      parts.push(...lines);
    }
    if (data && data.length > 0) {
      // First 10 rows of parsed tabular data
      for (let i = 0; i < Math.min(10, data.length); i++) {
        const row = data[i];
        if (Array.isArray(row)) {
          parts.push(row.map((cell) => String(cell || "")).join(" "));
        }
      }
    }
    return parts.join(" ").toLowerCase();
  });

  // Order existing bank/credit card/source accounts so that any match with statement header section is ranked first
  let orderedBaseAccounts = $derived.by(() => {
    // Primary candidates for statement sources are Assets:* and Liabilities:*
    const sourceCandidates = accountsList.filter(
      (a) => a.startsWith("Assets:") || a.startsWith("Liabilities:")
    );
    const otherAccounts = accountsList.filter(
      (a) => !a.startsWith("Assets:") && !a.startsWith("Liabilities:")
    );

    const allCandidates = [...sourceCandidates, ...otherAccounts];
    const header = statementHeaderContext;

    if (!header) {
      return allCandidates;
    }

    const scored = allCandidates.map((acc) => {
      let score = 0;
      // Extract tokens from account name (e.g., "Assets:Checking:ICICI" -> ["assets", "checking", "icici"])
      const tokens = acc
        .toLowerCase()
        .split(/[:\s_-]+/)
        .filter((t) => t.length >= 3 && t !== "assets" && t !== "liabilities");

      for (const tok of tokens) {
        if (header.includes(tok)) {
          score += 10;
        }
      }

      // Exact substring match for the full leaf name
      const leaf = acc.split(":").pop()?.toLowerCase();
      if (leaf && leaf.length >= 3 && header.includes(leaf)) {
        score += 25;
      }

      return { acc, score };
    });

    scored.sort((a, b) => b.score - a.score);
    return scored.map((s) => s.acc);
  });

  // Top suggested source account when a header match is found
  let topSuggestedBaseAccount = $derived.by(() => {
    if (!statementHeaderContext || orderedBaseAccounts.length === 0) return "";
    const first = orderedBaseAccounts[0];
    const tokens = first
      .toLowerCase()
      .split(/[:\s_-]+/)
      .filter((t) => t.length >= 3 && t !== "assets" && t !== "liabilities");
    const matched = tokens.some((t) => statementHeaderContext.includes(t));
    return matched ? first : "";
  });

  const saveAsNameDuplicate = $derived(
    !!_.find(templates, { name: saveAsName, template_type: "custom" })
  );
  const selectedPreviewCount = $derived(
    filterSelectedRows(previewRows, includedPreviewRows).length
  );

  async function save() {
    if (!templateEditor) return;
    const { template, saved, message } = await ajax("/api/templates/upsert", {
      method: "POST",
      body: JSON.stringify({
        name: saveAsName,
        content: templateEditor.state.doc.toString()
      }),
      background: true
    });

    if (!saved) {
      toast.toast({
        message: `Failed to save ${saveAsName}. reason: ${message}`,
        type: "is-danger",
        duration: 10000
      });
      return;
    }

    ({ templates } = await ajax("/api/templates", { background: true }));
    selectedTemplate = _.find(templates, { id: template.id });
    saveAsName = selectedTemplate?.name || "";
    toast.toast({
      message: `Saved ${saveAsName}`,
      type: "is-success"
    });

    $templateEditorState = _.assign({}, $templateEditorState, { hasUnsavedChanges: false });
  }

  async function remove() {
    if (!selectedTemplate) return;
    const oldName = selectedTemplate.name;
    const confirmed = confirm(`Are you sure you want to delete ${oldName} template?`);
    if (!confirmed) {
      return;
    }
    const { success, message } = await ajax("/api/templates/delete", {
      method: "POST",
      body: JSON.stringify({
        name: selectedTemplate.name
      }),
      background: true
    });

    if (!success) {
      toast.toast({
        message: `Failed to remove ${oldName}. reason: ${message}`,
        type: "is-danger",
        duration: 10000
      });
      return;
    }

    ({ templates } = await ajax("/api/templates", { background: true }));
    selectedTemplate = templates[0];
    saveAsName = selectedTemplate?.name || "";
    toast.toast({
      message: `Removed ${oldName}`,
      type: "is-success"
    });

    $templateEditorState = _.assign({}, $templateEditorState, { hasUnsavedChanges: false });
  }

  $effect(() => {
    if (!_.isEmpty(data) && $templateEditorState.template && previewEditor) {
      try {
        const selectedRows = filterSelectedRows(rows, includedPreviewRows);
        preview = renderJournal(selectedRows, $templateEditorState.template, {
          reverse: options.reverse,
          trim: options.trim
        });
        updatePreviewContent(previewEditor, preview);
      } catch (e) {
        console.log(e);
      }
    }
  });

  $effect(() => {
    if (selectedTemplate && templateEditor && templateEditorDom) {
      if (templateEditor.state.doc.toString() != selectedTemplate.content) {
        templateEditor.destroy();
        templateEditor = createTemplateEditor(selectedTemplate.content, templateEditorDom);
      }
    }
  });

  $effect(() => {
    if (selectedPreset) {
      delimiter = selectedPreset.delimiter || ",";
      if (!_.isEmpty(data)) {
        refreshImportPreview();
      }
    }
  });

  async function handleFilesSelect(detail: { acceptedFiles: File[] }) {
    const { acceptedFiles } = detail;
    if (!acceptedFiles || acceptedFiles.length === 0) return;

    const file = acceptedFiles[0];
    const ext = file.name.split(".").pop()?.toLowerCase();

    // OFX / QFX statement check
    if (ext === "ofx" || ext === "qfx") {
      rawFileContent = await file.text();
      detectedFormat = ext === "qfx" ? "qfx" : "ofx";
      mode = "smart";
      await runSmartIngest();
      return;
    }

    if (ext === "csv" || ext === "txt") {
      rawFileContent = await file.text();
      if (rawFileContent.includes("<OFX>") || rawFileContent.includes("<STMTTRN>")) {
        detectedFormat = "ofx";
        mode = "smart";
        await runSmartIngest();
        return;
      }
      detectedFormat = "csv";
    } else if (ext === "xls" || ext === "xlsx") {
      detectedFormat = "excel";
      rawFileContent = "";
    } else if (ext === "pdf") {
      detectedFormat = "pdf";
      rawFileContent = "";
    } else {
      detectedFormat = "csv";
      rawFileContent = await file.text();
    }

    // Standard tabular parsing (CSV, XLS, XLSX, PDF via parse())
    const results = await parse(file);
    if (results.error) {
      parseErrorMessage = results.error;
    } else {
      parseErrorMessage = null;
      data = results.data;
      rows = asRows(results);

      const maxRow = _.maxBy(data, (row) => row.length);
      columnCount = maxRow ? maxRow.length : 0;
      _.each(data, (row) => {
        row.length = columnCount;
      });

      if (mode === "smart") {
        await runSmartIngest();
      } else {
        await refreshImportPreview();
      }
    }
  }

  async function runSmartIngest() {
    if (!rawFileContent && _.isEmpty(data)) return;

    let contentToIngest = rawFileContent;
    if (
      (detectedFormat === "csv" || detectedFormat === "excel" || detectedFormat === "pdf") &&
      !_.isEmpty(data)
    ) {
      contentToIngest = toCSVContent(data);
    }

    try {
      const response: IngestionResult = await ajax("/api/import/statement", {
        method: "POST",
        body: JSON.stringify({
          format: detectedFormat,
          content: contentToIngest,
          delimiter,
          base_account: baseAccount
        }),
        background: true
      });

      stagedTransactions = (response.transactions || []).map((tx) => {
        // If not already categorized by a user rule or backend suggester, predict using TF-IDF cosine similarity
        if (!tx.selected_account && !tx.suggested_account && !tx.is_duplicate) {
          const query = [tx.payee, tx.memo].filter(Boolean).join(" ");
          const matches = findMatch(query);
          if (matches && matches.length > 0) {
            // Filter by logical account type:
            // For Debits (outflow): prioritize Expenses:* at top, then Liabilities:*/Assets:*, strictly deprioritizing Equity/Income
            // For Credits (inflow): prioritize Income:* at top, then Assets:*/Liabilities:*, strictly deprioritizing Expenses/Equity
            let preferredMatch = tx.is_debit
              ? matches.find(([acc, score]) => score > 0.02 && acc.startsWith("Expenses:")) ||
                matches.find(
                  ([acc, score]) =>
                    score > 0.03 && (acc.startsWith("Liabilities:") || acc.startsWith("Assets:"))
                )
              : matches.find(([acc, score]) => score > 0.02 && acc.startsWith("Income:")) ||
                matches.find(
                  ([acc, score]) =>
                    score > 0.03 && (acc.startsWith("Assets:") || acc.startsWith("Liabilities:"))
                );

            if (!preferredMatch && matches[0][1] > 0.05) {
              if (
                tx.is_debit &&
                !matches[0][0].startsWith("Equity:") &&
                !matches[0][0].startsWith("Income:")
              ) {
                preferredMatch = matches[0];
              } else if (
                !tx.is_debit &&
                !matches[0][0].startsWith("Equity:") &&
                !matches[0][0].startsWith("Expenses:")
              ) {
                preferredMatch = matches[0];
              }
            }

            if (preferredMatch) {
              const bestAcc = preferredMatch[0];
              return {
                ...tx,
                suggested_account: bestAcc,
                selected_account: "", // Requires user acceptance or edit
                confidence: "ai_suggested"
              };
            }
          }
        }
        return tx;
      });

      // Default include only non-duplicate transactions that have already been assigned (or rule matched)
      includedStaged = stagedTransactions.map((tx) => !tx.is_duplicate && !!tx.selected_account);

      const suggestedCount = stagedTransactions.filter(
        (t) => !t.selected_account && !!t.suggested_account
      ).length;
      toast.toast({
        message: `Parsed ${response.total_parsed} transactions (${response.auto_categorized} rule-matched, ${suggestedCount} suggested, ${response.possible_duplicates} duplicates flagged)`,
        type: "is-info",
        duration: 4000
      });
    } catch (e: any) {
      parseErrorMessage = e.message;
    }
  }

  function extractShortNote(payee?: string, memo?: string): string {
    return cleanPayee(payee) || cleanPayee(memo) || (payee || memo || "").slice(0, 32).trim();
  }

  function handleCreateRuleFromTransaction(tx: StagedTransaction) {
    const payeeClean = extractShortNote(tx.payee, tx.memo);

    initialRuleForStudio = {
      name: `Auto: ${payeeClean || tx.payee}`,
      payee_pattern: payeeClean || tx.payee,
      payee_match_type: "contains",
      tx_type: tx.is_debit ? "debit" : "credit",
      target_account: tx.selected_account || accountsList[0] || "Expenses:Miscellaneous",
      priority: 100
    };
    ruleStudioOpen = true;
  }

  async function handleQuickSaveRule(tx: StagedTransaction) {
    if (!tx.selected_account) return;
    const payeeClean = extractShortNote(tx.payee, tx.memo) || tx.payee;

    try {
      const res: any = await ajax("/api/rules", {
        method: "POST",
        body: JSON.stringify({
          name: `Rule: ${payeeClean}`,
          payee_pattern: payeeClean,
          payee_match_type: "contains",
          tx_type: tx.is_debit ? "debit" : "credit",
          target_account: tx.selected_account,
          priority: 100,
          enabled: true
        }),
        background: true
      });

      if (res?.saved) {
        toast.toast({
          message: `Saved rule for "${payeeClean}" -> ${tx.selected_account}`,
          type: "is-success",
          duration: 3500
        });

        // Re-run smart ingest or update matching transactions in staging table
        stagedTransactions.forEach((t) => {
          if (!t.selected_account && t.payee.toLowerCase().includes(payeeClean.toLowerCase())) {
            t.selected_account = tx.selected_account;
            t.matched_rule_name = res.rule?.name || `Rule: ${payeeClean}`;
            t.confidence = "rule";
          }
        });
      }
    } catch (e: any) {
      toast.toast({
        message: `Failed to save rule: ${e.message}`,
        type: "is-danger",
        duration: 5000
      });
    }
  }

  function generateLedgerText(selected: StagedTransaction[], clubByDate: boolean): string {
    const lines: string[] = [];

    const writeSingleTx = (tx: StagedTransaction) => {
      const dateFormatted = tx.date.replaceAll("-", "/");
      const shortNote = extractShortNote(tx.payee, tx.memo) || "Transaction";
      const curr = (tx.currency || defaultCurrency || "INR").trim();
      const amtStr = `${tx.amount.toFixed(2)} ${curr}`;
      const tagStr = tx.tags && tx.tags.length > 0 ? `  ; ${tx.tags.join(" ")}` : "";

      lines.push(`${dateFormatted} * ${shortNote}${tagStr}`);
      if (tx.is_debit) {
        lines.push(`    ${baseAccount.padEnd(40)}  -${amtStr}`);
        lines.push(`    ${(tx.selected_account || "Expenses:Unknown").padEnd(40)}   ${amtStr}`);
      } else {
        lines.push(`    ${baseAccount.padEnd(40)}   ${amtStr}`);
        lines.push(`    ${(tx.selected_account || "Income:Unknown").padEnd(40)}  -${amtStr}`);
      }
      lines.push("");
    };

    if (clubByDate) {
      const debitGroups = new Map<string, StagedTransaction[]>();
      const creditTxs: StagedTransaction[] = [];

      for (const tx of selected) {
        const curr = (tx.currency || defaultCurrency || "INR").trim();
        if (tx.is_debit) {
          const key = `${tx.date}__${curr}`;
          if (!debitGroups.has(key)) {
            debitGroups.set(key, []);
          }
          debitGroups.get(key)!.push(tx);
        } else {
          creditTxs.push(tx);
        }
      }

      for (const [key, txs] of debitGroups.entries()) {
        if (txs.length === 1) {
          writeSingleTx(txs[0]);
          continue;
        }
        const [date, curr] = key.split("__");
        const dateFormatted = date.replaceAll("-", "/");
        lines.push(`${dateFormatted} * Consolidated Debits (${txs.length} items)`);
        let total = 0;
        for (const tx of txs) {
          total += tx.amount;
          const acc = (tx.selected_account || "Expenses:Unknown").padEnd(40);
          const amtStr = `${tx.amount.toFixed(2)} ${curr}`;
          let note = extractShortNote(tx.payee, tx.memo);
          if (tx.tags && tx.tags.length > 0) {
            note = note ? `${note} (${tx.tags.join(" ")})` : tx.tags.join(" ");
          }
          lines.push(`    ${acc}   ${amtStr}${note ? `  ; ${note}` : ""}`);
        }
        lines.push(`    ${baseAccount.padEnd(40)}  -${total.toFixed(2)} ${curr}`);
        lines.push("");
      }

      for (const tx of creditTxs) {
        writeSingleTx(tx);
      }
    } else {
      for (const tx of selected) {
        writeSingleTx(tx);
      }
    }

    return lines.join("\n");
  }

  function openReviewModal() {
    if (!baseAccount.trim()) {
      toast.toast({
        message: "Please select your Source / Bank Account first.",
        type: "is-warning"
      });
      return;
    }

    const selected = stagedTransactions.filter((_, idx) => includedStaged[idx]);
    if (selected.length === 0) {
      toast.toast({ message: "No transactions selected to review", type: "is-warning" });
      return;
    }

    reviewLedgerContent = generateLedgerText(selected, groupByDate);
    reviewLedgerOpen = true;
  }

  async function commitStagedTransactions(useRawReview = false) {
    if (!baseAccount.trim() && !useRawReview) {
      toast.toast({
        message: "Please select your Source / Bank Account first.",
        type: "is-warning"
      });
      return;
    }

    const selected = stagedTransactions.filter((_, idx) => includedStaged[idx]);
    if (selected.length === 0 && (!useRawReview || !reviewLedgerContent.trim())) {
      toast.toast({ message: "No transactions selected to commit", type: "is-warning" });
      return;
    }

    importSaving = true;
    try {
      const res: any = await ajax("/api/import/commit", {
        method: "POST",
        body: JSON.stringify({
          base_account: baseAccount,
          commodity: defaultCurrency,
          group_by_date: groupByDate,
          raw_content: useRawReview ? reviewLedgerContent : "",
          transactions: selected
        }),
        background: true
      });

      if (res.success) {
        toast.toast({
          message: `Successfully appended ${res.committed_count} transactions to ledger!`,
          type: "is-success",
          duration: 6000
        });

        // Remove committed transactions from staged list
        stagedTransactions = stagedTransactions.filter((_, idx) => !includedStaged[idx]);
        includedStaged = stagedTransactions.map(() => true);
        reviewLedgerOpen = false;
      }
    } catch (e: any) {
      toast.toast({ message: `Failed to commit transactions: ${e.message}`, type: "is-danger" });
    } finally {
      importSaving = false;
    }
  }

  async function refreshImportPreview() {
    if (_.isEmpty(data) || !selectedTemplate) {
      return;
    }
    const response: any = await ajax("/api/import/preview", {
      method: "POST",
      body: JSON.stringify({
        template: selectedTemplate.name,
        content: toCSVContent(data),
        delimiter,
        dry_run: true
      }),
      background: true
    });

    if (response.error) {
      parseErrorMessage = response.error.message;
      previewRows = [];
      includedPreviewRows = [];
      return;
    }

    parseErrorMessage = null;
    previewRows = response.rows;
    includedPreviewRows = defaultIncludedFromValidation(previewRows);
  }

  async function copyToClipboard() {
    try {
      await navigator.clipboard.writeText(preview);
      toast.toast({
        message: "Copied to clipboard",
        type: "is-success"
      });
    } catch (e) {
      console.log(e);
      toast.toast({
        message: "Failed to copy to clipboard",
        type: "is-danger"
      });
    }
  }

  let modalOpen = $state(false);
  function openSaveModal() {
    if (selectedPreviewCount === 0) {
      toast.toast({
        message: "Select at least one row to import",
        type: "is-danger"
      });
      return;
    }
    modalOpen = true;
  }

  async function saveToFile(destinationFile: string) {
    importSaving = true;
    const { saved, message } = await ajax("/api/editor/save", {
      method: "POST",
      body: JSON.stringify({ name: destinationFile, content: preview, operation: "overwrite" }),
      background: true
    });
    importSaving = false;

    if (saved) {
      toast.toast({
        message: `Saved <b><a href="/ledger/editor/${encodeURIComponent(
          destinationFile
        )}">${destinationFile}</a></b>`,
        type: "is-success",
        duration: 5000
      });
    } else {
      toast.toast({
        message: `Failed to save ${destinationFile}. reason: ${message}`,
        type: "is-danger",
        duration: 10000
      });
    }
  }

  async function saveCurrentAsPreset() {
    const name = prompt("Preset name");
    if (_.isEmpty(name)) {
      return;
    }

    const columnMappings = _.fromPairs(
      _.range(0, columnCount).map((index) => {
        const col = String.fromCharCode(65 + index);
        return [col, col];
      })
    );

    const response: any = await ajax("/api/import/presets", {
      method: "POST",
      body: JSON.stringify({
        name,
        column_mappings: columnMappings,
        date_format: "",
        default_accounts: {},
        delimiter
      }),
      background: true
    });

    if (response.error || !response.saved) {
      toast.toast({
        message: `Failed to save preset ${name}`,
        type: "is-danger"
      });
      return;
    }

    ({ presets: importPresets } = await ajax("/api/import/presets", { background: true }));
    selectedPreset = _.find(importPresets, { id: response.preset.id });
    toast.toast({
      message: `Saved preset ${name}`,
      type: "is-success"
    });
  }

  function builtinNotAllowed(action: string, template: ImportTemplate | null) {
    if (template?.template_type == "builtin") {
      return `Not allowed to ${action.toLowerCase()} builtin template`;
    }
    return action;
  }

  let templateCreateModalOpen = $state(false);
  function openTemplateCreateModal() {
    templateCreateModalOpen = true;
  }
</script>

<Modal bind:active={templateCreateModalOpen}>
  {#snippet head(close)}
    <p class="text-base font-semibold flex-1">Create Template</p>
    <button
      class="du-btn du-btn-sm du-btn-circle du-btn-ghost"
      aria-label="close"
      onclick={() => close()}
    >
      <i class="fas fa-times" aria-hidden="true"></i>
    </button>
  {/snippet}
  {#snippet body()}
    <div class="field">
      <label class="label" for="save-filename">Template Name</label>
      <div class="control" id="save-filename">
        <input class="input" type="text" bind:value={saveAsName} />
        {#if saveAsNameDuplicate}
          <p class="help is-danger">Template with the same name already exists</p>
        {/if}
      </div>
    </div>
  {/snippet}
  {#snippet foot(close)}
    <button
      class="du-btn du-btn-success du-btn-sm"
      disabled={_.isEmpty(saveAsName) || saveAsNameDuplicate}
      onclick={() => {
        save();
        close();
      }}>Create</button
    >
    <button class="du-btn du-btn-sm" onclick={() => close()}>Cancel</button>
  {/snippet}
</Modal>

<FileModal bind:open={modalOpen} onsave={(file) => saveToFile(file)} />
<RuleStudioModal
  bind:open={ruleStudioOpen}
  accounts={accountsList}
  initialRule={initialRuleForStudio}
  onruleschanged={() => {
    initialRuleForStudio = null;
    if (stagedTransactions.length > 0) {
      runSmartIngest();
    }
  }}
/>

<!-- Review & Edit Generated Ledger Modal -->
<Modal bind:active={reviewLedgerOpen}>
  {#snippet head(close)}
    <p class="text-base font-semibold flex-1">
      <i class="fas fa-file-invoice mr-2 has-text-primary"></i>
      Review Generated Ledger Entries
    </p>
    <button
      class="du-btn du-btn-sm du-btn-circle du-btn-ghost"
      aria-label="close"
      onclick={() => close()}
    >
      <i class="fas fa-times" aria-hidden="true"></i>
    </button>
  {/snippet}
  {#snippet body()}
    <div class="is-flex is-justify-content-space-between is-align-items-center mb-2">
      <p class="is-size-7 has-text-grey mb-0">
        Review and optionally edit the generated transactions before committing them into your
        journal.
      </p>
      <label class="checkbox is-size-7 has-text-weight-medium">
        <input
          type="checkbox"
          bind:checked={groupByDate}
          onchange={() => {
            const selected = stagedTransactions.filter((_, idx) => includedStaged[idx]);
            reviewLedgerContent = generateLedgerText(selected, groupByDate);
          }}
        />
        Club debits by date
      </label>
    </div>
    <div class="field">
      <div class="control">
        <textarea
          class="textarea is-small font-mono"
          style="min-height: 320px; white-space: pre; font-size: 0.85rem;"
          bind:value={reviewLedgerContent}
        ></textarea>
      </div>
    </div>
  {/snippet}
  {#snippet foot(close)}
    <button
      class="du-btn du-btn-success du-btn-sm"
      class:loading={importSaving}
      disabled={importSaving || _.isEmpty(reviewLedgerContent.trim())}
      onclick={() => commitStagedTransactions(true)}
    >
      <i class="fas fa-check-double mr-1"></i>
      Confirm & Append to Ledger
    </button>
    <button class="du-btn du-btn-sm" onclick={() => close()}>Cancel</button>
  {/snippet}
</Modal>

<section class="section tab-import" style="padding-bottom: 0 !important">
  <div class="container is-fluid">
    <!-- Pipeline Header & Mode Switcher -->
    <div class="is-flex is-justify-content-space-between is-align-items-center mb-4">
      <div class="tabs is-toggle is-toggle-rounded is-small mb-0">
        <ul>
          <li class:is-active={mode === "smart"}>
            <a
              href={"#"}
              onclick={(e) => {
                e.preventDefault();
                mode = "smart";
              }}
            >
              <span class="icon is-small"><i class="fas fa-wand-magic-sparkles"></i></span>
              <span>Smart Ingestion & Reconciliation</span>
            </a>
          </li>
          <li class:is-active={mode === "template"}>
            <a
              href={"#"}
              onclick={(e) => {
                e.preventDefault();
                mode = "template";
              }}
            >
              <span class="icon is-small"><i class="fas fa-code"></i></span>
              <span>Classic Template Editor</span>
            </a>
          </li>
        </ul>
      </div>

      <div class="buttons are-small mb-0">
        <button class="button is-primary is-light" onclick={() => (ruleStudioOpen = true)}>
          <span class="icon is-small"><i class="fas fa-sliders"></i></span>
          <span>Rule Studio</span>
        </button>
      </div>
    </div>

    {#if mode === "smart"}
      <div class="columns mb-0">
        <div class="column is-4 py-0">
          <div class="box p-3 mb-3">
            <h4 class="title is-6 mb-2">1. Upload Statement</h4>
            <Dropzone
              multiple={false}
              accept=".csv,.txt,.ofx,.qfx,.xls,.xlsx,.pdf,.CSV,.TXT,.OFX,.QFX,.XLS,.XLSX,.PDF"
              ondrop={handleFilesSelect}
            >
              Drag 'n' drop CSV, OFX, QFX, Excel (.xlsx, .xls) or PDF statement here
            </Dropzone>
            {#if detectedFormat}
              <div class="mt-2 is-size-7 has-text-grey">
                Detected format: <span class="tag is-small is-light uppercase"
                  >{detectedFormat}</span
                >
              </div>
            {/if}
          </div>

          <div class="box p-3 mb-3">
            <h4 class="title is-6 mb-2">2. Statement Account</h4>
            <div class="field mb-3">
              <label class="label is-small" for="smart-base-account">Source / Bank Account</label>
              <div class="control">
                <input
                  id="smart-base-account"
                  class="input is-small {!baseAccount ? 'is-warning' : ''}"
                  type="text"
                  list="accounts-datalist"
                  bind:value={baseAccount}
                  placeholder={topSuggestedBaseAccount
                    ? `Suggested: ${topSuggestedBaseAccount}`
                    : "Select source account (e.g. Assets:Checking:ICICI)..."}
                />
                <datalist id="accounts-datalist">
                  {#each orderedBaseAccounts as acc}
                    <option value={acc}>{acc}</option>
                  {/each}
                </datalist>
              </div>
              {#if topSuggestedBaseAccount && !baseAccount}
                <div class="is-flex is-align-items-center mt-1">
                  <span class="is-size-7 has-text-grey mr-2 truncate">
                    <i class="fas fa-lightbulb has-text-warning mr-1"></i>
                    Header match:
                    <strong class="has-text-grey-darker">{topSuggestedBaseAccount}</strong>
                  </span>
                  <button
                    class="button is-small is-link is-light py-0 px-2 ml-auto"
                    style="height: 20px; font-size: 0.7rem;"
                    onclick={() => {
                      baseAccount = topSuggestedBaseAccount;
                    }}
                  >
                    Select
                  </button>
                </div>
              {/if}
              <p class="help is-size-7">The physical account whose statement you are importing.</p>
            </div>

            <div class="field is-grouped is-align-items-center mb-0">
              <label class="label is-small mb-0 mr-2" for="smart-delimiter">Delimiter</label>
              <div class="control">
                <input
                  id="smart-delimiter"
                  class="input is-small"
                  style="width: 50px;"
                  maxlength="1"
                  bind:value={delimiter}
                  onblur={() => runSmartIngest()}
                />
              </div>
              <button
                class="button is-small ml-auto"
                disabled={_.isEmpty(data) && !rawFileContent}
                onclick={() => runSmartIngest()}
              >
                Re-analyze
              </button>
            </div>
          </div>

          <div class="box p-3">
            <h4 class="title is-6 mb-2">3. Review & Append to Ledger</h4>
            <p class="is-size-7 has-text-grey mb-2">
              Review generated entries before appending, or commit directly with duplicate safety
              checks.
            </p>
            <div class="field mb-3">
              <label class="checkbox is-size-7">
                <input type="checkbox" bind:checked={groupByDate} />
                Club multiple debit entries per date into single transaction
              </label>
            </div>
            <div class="buttons mb-0">
              <button
                class="button is-link is-light is-fullwidth mb-2"
                disabled={stagedTransactions.length === 0 ||
                  stagedTransactions.filter((_, i) => includedStaged[i]).length === 0}
                onclick={openReviewModal}
              >
                <span class="icon is-small"><i class="fas fa-file-invoice"></i></span>
                <span
                  >Review / Edit Ledger ({stagedTransactions.filter((_, i) => includedStaged[i])
                    .length})</span
                >
              </button>
              <button
                class="button is-success is-fullwidth"
                class:is-loading={importSaving}
                disabled={stagedTransactions.length === 0 ||
                  stagedTransactions.filter((_, i) => includedStaged[i]).length === 0 ||
                  importSaving}
                onclick={() => commitStagedTransactions(false)}
              >
                <span class="icon is-small"><i class="fas fa-check-double"></i></span>
                <span
                  >Quick Commit ({stagedTransactions.filter((_, i) => includedStaged[i])
                    .length})</span
                >
              </button>
            </div>
          </div>
        </div>

        <div class="column is-8 py-0">
          {#if parseErrorMessage}
            <div class="message invertable is-danger mb-3">
              <div class="message-header">Ingestion Warning</div>
              <div class="message-body">{parseErrorMessage}</div>
            </div>
          {/if}

          {#if stagedTransactions.length > 0}
            <TransactionStagingTable
              bind:transactions={stagedTransactions}
              bind:included={includedStaged}
              accounts={accountsList}
              {defaultCurrency}
              oneditrule={handleCreateRuleFromTransaction}
              onquicksave={handleQuickSaveRule}
            />
          {:else}
            <div class="notification is-light has-text-centered py-6">
              <span class="icon is-large has-text-grey-light mb-2">
                <i class="fas fa-file-invoice-dollar fa-3x"></i>
              </span>
              <p class="has-text-grey is-size-6 font-semibold">No Statement Uploaded Yet</p>
              <p class="is-size-7 has-text-grey">
                Drag 'n' drop your bank statement (CSV, OFX, QFX, Excel, PDF) into the upload box on
                the left to start auto-categorizing.
              </p>
            </div>
          {/if}
        </div>
      </div>
    {:else}
      <!-- Classic Template Mode -->
      <div class="columns mb-0">
        <div class="column is-5 py-0">
          <div class="box p-3 mb-3 overflow-x-auto">
            <div class="field is-grouped mb-0">
              <p class="control">
                <span data-tippy-content="Create" data-tippy-followCursor="false">
                  <button
                    class="button"
                    aria-label="Create template"
                    onclick={() => openTemplateCreateModal()}
                  >
                    <span class="icon">
                      <i class="fas fa-file-circle-plus"></i>
                    </span>
                  </button>
                </span>

                <span
                  class="ml-4"
                  data-tippy-followCursor="false"
                  data-tippy-content={$templateEditorState.hasUnsavedChanges == false
                    ? "No Unsaved Chagnes"
                    : builtinNotAllowed("Save", selectedTemplate)}
                >
                  <button
                    class="button"
                    aria-label="Save template"
                    onclick={() => save()}
                    disabled={$templateEditorState.hasUnsavedChanges == false ||
                      selectedTemplate?.template_type == "builtin"}
                  >
                    <span class="icon">
                      <i class="fas fa-floppy-disk"></i>
                    </span>
                  </button>
                </span>

                <span
                  data-tippy-followCursor="false"
                  data-tippy-content={builtinNotAllowed("Delete", selectedTemplate)}
                >
                  <button
                    class="button"
                    aria-label="Delete template"
                    onclick={() => remove()}
                    disabled={selectedTemplate?.template_type == "builtin"}
                  >
                    <span class="icon">
                      <i class="fas fa-trash-can"></i>
                    </span>
                  </button>
                </span>
              </p>

              <p class="control is-expanded">
                <Select
                  bind:value={selectedTemplate}
                  showChevron={true}
                  items={templates}
                  label="name"
                  itemId="id"
                  searchable={true}
                  clearable={false}
                  floatingConfig={{ strategy: "fixed" }}
                  on:change={() => {
                    if (selectedTemplate) {
                      saveAsName = selectedTemplate.name;
                    }
                  }}
                >
                  <div slot="selection" let:selection>
                    {#if selection}
                      {selection.name}
                      <span class="tag is-small is-link invertable is-light"
                        >{selection.template_type}</span
                      >
                    {/if}
                  </div>
                  <div slot="item" let:item>
                    <span class="name">{item.name}</span>
                    <span class="tag is-small is-link invertable is-light"
                      >{item.template_type}</span
                    >
                  </div>
                </Select>
              </p>
            </div>
          </div>
          <div class="box py-0">
            <div class="field">
              <div class="control">
                <div class="template-editor" bind:this={templateEditorDom}></div>
              </div>
            </div>
          </div>
          <div class="box py-0">
            <div class="field">
              <div class="control">
                <button
                  data-tippy-followCursor="false"
                  data-tippy-content="Copy to Clipboard"
                  class="button clipboard"
                  aria-label="Copy preview to clipboard"
                  disabled={_.isEmpty(preview)}
                  onclick={copyToClipboard}
                >
                  <span class="icon">
                    <i class="fas fa-copy"></i>
                  </span>
                </button>
                <button
                  data-tippy-followCursor="false"
                  data-tippy-content="Confirm & Save"
                  class="button save"
                  class:is-loading={importSaving}
                  aria-label="Confirm selected rows and save"
                  disabled={_.isEmpty(preview) || selectedPreviewCount === 0 || importSaving}
                  onclick={openSaveModal}
                >
                  <span class="icon">
                    <i class="fas fa-floppy-disk"></i>
                  </span>
                </button>
                <div class="preview-editor" bind:this={previewEditorDom}></div>
              </div>
            </div>
          </div>
        </div>
        <div class="column is-7 py-0">
          <div class="box p-3 mb-3">
            <Dropzone
              multiple={false}
              accept=".csv,.txt,.xls,.xlsx,.pdf,.CSV,.TXT,.XLS,.XLSX,.PDF"
              ondrop={handleFilesSelect}
            >
              Drag 'n' drop CSV, TXT, XLS, XLSX, PDF file here or click to select
            </Dropzone>
          </div>
          <div class="box p-3 mb-3">
            <PresetSelector
              presets={importPresets}
              bind:selectedPreset
              onsavecurrent={saveCurrentAsPreset}
            />
            <div class="field is-grouped is-align-items-center">
              <label class="label mb-0 mr-2" for="import-delimiter">Delimiter</label>
              <div class="control">
                <input
                  id="import-delimiter"
                  class="input is-small"
                  maxlength="1"
                  bind:value={delimiter}
                  onblur={() => refreshImportPreview()}
                />
              </div>
            </div>
          </div>
          <div class="is-flex justify-end mb-3 gap-4">
            <div class="field color-switch">
              <input
                id="import-reverse"
                type="checkbox"
                bind:checked={options.reverse}
                class="switch is-rounded is-small"
              />
              <label for="import-reverse">Reverse</label>
            </div>
            <div class="field color-switch">
              <input
                id="trim-reverse"
                type="checkbox"
                bind:checked={options.trim}
                class="switch is-rounded is-small"
              />
              <label for="trim-reverse">Trim</label>
            </div>
          </div>
          {#if parseErrorMessage}
            <div class="message invertable is-danger">
              <div class="message-header">Failed to parse document</div>
              <div class="message-body">{parseErrorMessage}</div>
            </div>
          {/if}
          {#if !_.isEmpty(data)}
            <ImportPreviewTable rows={previewRows} bind:included={includedPreviewRows} />
          {/if}
        </div>
      </div>
    {/if}
    <div></div>
  </div>
</section>

<style lang="scss">
  @import "bulma/sass/utilities/_all.sass";

  $import-full-height: calc(100vh - 205px);

  .clipboard {
    float: right;
    position: absolute;
    right: 0;
    z-index: 1;
  }

  .save {
    float: right;
    position: absolute !important;
    right: 40px;
    z-index: 1;
  }

  .color-switch {
    .switch[type="checkbox"]:checked + label::before,
    .switch[type="checkbox"]:checked + label:before {
      background: $link;
    }
  }
</style>
