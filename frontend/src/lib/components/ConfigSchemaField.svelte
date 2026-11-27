<script lang="ts">
  import { ChevronDown, ChevronRight, Plus, Trash2 } from "lucide-svelte";
  import ConfigSchemaField from "./ConfigSchemaField.svelte";

  export interface JsonSchemaNode {
    type?: string | string[];
    title?: string;
    description?: string;
    default?: unknown;
    enum?: unknown[];
    properties?: Record<string, JsonSchemaNode>;
    additionalProperties?: boolean | JsonSchemaNode;
    items?: JsonSchemaNode;
    required?: string[];
    "ui:header"?: string;
    "ui:widget"?: string;
    "ui:order"?: number;
    "ui:open"?: boolean;
  }

  let {
    fieldKey,
    value = $bindable<unknown>(),
    schema,
    accounts = [],
    depth = 0,
    required = false,
    readonly = false,
    onValueChange,
    onDelete
  }: {
    fieldKey: string;
    value: unknown;
    schema: JsonSchemaNode;
    accounts?: string[];
    depth?: number;
    required?: boolean;
    readonly?: boolean;
    onValueChange?: (next: unknown) => void;
    onDelete?: () => void;
  } = $props();

  function setVal(next: unknown) {
    if (onValueChange) {
      onValueChange(next);
    } else {
      value = next;
    }
  }

  let expanded = $state(false);
  let expandedInit = false;
  $effect(() => {
    if (!expandedInit && schema) {
      expanded = schema["ui:open"] ?? depth < 1;
      expandedInit = true;
    }
  });

  let newMapKey = $state("");

  const label = $derived(
    schema?.title ||
      fieldKey
        .replace(/[_-]+/g, " ")
        .replace(/([a-z])([A-Z])/g, "$1 $2")
        .replace(/\b\w/g, (c) => c.toUpperCase())
  );

  const schemaType = $derived(Array.isArray(schema?.type) ? schema.type[0] : schema?.type);

  function defaultForSchema(node?: JsonSchemaNode): unknown {
    if (!node) return "";
    if (node.default !== undefined) return JSON.parse(JSON.stringify(node.default));
    const t = Array.isArray(node.type) ? node.type[0] : node.type;
    if (t === "boolean") return false;
    if (t === "number" || t === "integer") return 0;
    if (t === "array") return [];
    if (t === "object") return {};
    return "";
  }

  function sortedEntries(props: Record<string, JsonSchemaNode> = {}, req: string[] = []) {
    return Object.entries(props).sort(([ka, sa], [kb, sb]) => {
      const oa = sa["ui:order"] ?? 999;
      const ob = sb["ui:order"] ?? 999;
      if (oa !== ob) return oa - ob;
      const ra = req.includes(ka) ? 0 : 1;
      const rb = req.includes(kb) ? 0 : 1;
      if (ra !== rb) return ra - rb;
      const ta = sa.type === "object" ? 2 : sa.type === "array" ? 3 : 1;
      const tb = sb.type === "object" ? 2 : sb.type === "array" ? 3 : 1;
      if (ta !== tb) return ta - tb;
      return ka.localeCompare(kb);
    });
  }

  const objectValue = $derived(
    value && typeof value === "object" && !Array.isArray(value)
      ? (value as Record<string, unknown>)
      : ({} as Record<string, unknown>)
  );

  const arrayValue = $derived(Array.isArray(value) ? value : ([] as unknown[]));
</script>

{#if schemaType === "boolean"}
  <div class="flex items-start justify-between gap-4 py-2 border-b border-[var(--border)]/40 last:border-b-0">
    <div class="space-y-0.5">
      <label class="text-xs font-medium text-[var(--text)] cursor-pointer" for={`cfg-${fieldKey}-${depth}`}>
        {label}
      </label>
      {#if schema.description}
        <p class="text-[11px] text-[var(--muted)]">{schema.description}</p>
      {/if}
    </div>
    <input
      id={`cfg-${fieldKey}-${depth}`}
      type="checkbox"
      disabled={readonly}
      checked={Boolean(value)}
      onchange={(e) => setVal((e.currentTarget as HTMLInputElement).checked)}
      class="mt-0.5 h-4 w-4 accent-[var(--accent)] cursor-pointer"
    />
  </div>
{:else if schemaType === "object" && schema.properties}
  {@const obj = objectValue}
  {@const entries = sortedEntries(schema.properties, schema.required || [])}
  <div class="rounded-lg border border-[var(--border)] bg-[var(--surface-elevated)]/25 overflow-hidden">
    <div class="px-3.5 py-2.5 bg-[var(--surface-elevated)]/60 flex items-center justify-between gap-2">
      <button
        type="button"
        onclick={() => (expanded = !expanded)}
        class="flex items-center gap-1.5 text-xs font-semibold text-left cursor-pointer flex-1"
      >
        {#if expanded}
          <ChevronDown class="w-3.5 h-3.5 text-[var(--muted)]" />
        {:else}
          <ChevronRight class="w-3.5 h-3.5 text-[var(--muted)]" />
        {/if}
        <span>{label}</span>
        {#if schema["ui:header"] && obj[schema["ui:header"]]}
          <span class="text-[11px] font-normal text-[var(--muted)] ml-1">
            — {String(obj[schema["ui:header"]])}
          </span>
        {/if}
      </button>
      {#if onDelete && !readonly}
        <button
          type="button"
          onclick={onDelete}
          title="Remove"
          class="p-1 text-[var(--muted)] hover:text-[var(--loss)] cursor-pointer"
        >
          <Trash2 class="w-3.5 h-3.5" />
        </button>
      {/if}
    </div>
    {#if expanded}
      <div class="p-3.5 space-y-3">
        {#if schema.description}
          <p class="text-[11px] text-[var(--muted)]">{schema.description}</p>
        {/if}
        {#each entries as [subKey, subSchema] (subKey)}
          <ConfigSchemaField
            fieldKey={subKey}
            value={obj[subKey]}
            schema={subSchema}
            {accounts}
            depth={depth + 1}
            required={(schema.required || []).includes(subKey)}
            {readonly}
            onValueChange={(nextVal) => {
              setVal({ ...obj, [subKey]: nextVal });
            }}
          />
        {/each}
      </div>
    {/if}
  </div>
{:else if schemaType === "object" && schema.additionalProperties && typeof schema.additionalProperties === "object"}
  {@const mapObj = objectValue}
  {@const mapEntries = Object.entries(mapObj)}
  {@const valSchema = schema.additionalProperties as JsonSchemaNode}
  <div class="rounded-lg border border-[var(--border)] bg-[var(--surface-elevated)]/20 p-3.5 space-y-3">
    <div class="flex flex-wrap items-center justify-between gap-2">
      <div>
        <div class="text-xs font-semibold">{label} ({mapEntries.length})</div>
        {#if schema.description}
          <p class="text-[11px] text-[var(--muted)]">{schema.description}</p>
        {/if}
      </div>
      {#if !readonly}
        <div class="flex items-center gap-1.5">
          <input
            type="text"
            bind:value={newMapKey}
            placeholder="New key / name…"
            list={fieldKey === "accounts" ? "cfg-accounts-datalist" : undefined}
            class="px-2.5 py-1 rounded text-xs bg-[var(--surface)] border border-[var(--border)] outline-none w-44"
          />
          <button
            type="button"
            disabled={!newMapKey.trim()}
            onclick={() => {
              const k = newMapKey.trim();
              if (!k || k in mapObj) return;
              setVal({ ...mapObj, [k]: defaultForSchema(valSchema) });
              newMapKey = "";
            }}
            class="inline-flex items-center gap-1 px-2.5 py-1 rounded bg-[var(--accent)] text-[var(--accent-fg)] text-xs font-medium disabled:opacity-40 cursor-pointer"
          >
            <Plus class="w-3.5 h-3.5" />
            <span>Add</span>
          </button>
        </div>
      {/if}
    </div>

    {#if mapEntries.length === 0}
      <div class="text-xs text-[var(--muted)] py-2">No entries configured.</div>
    {:else}
      <div class="space-y-2">
        {#each mapEntries as [entryKey] (entryKey)}
          <ConfigSchemaField
            fieldKey={entryKey}
            value={mapObj[entryKey]}
            schema={valSchema}
            {accounts}
            depth={depth + 1}
            {readonly}
            onValueChange={(nextVal) => {
              setVal({ ...mapObj, [entryKey]: nextVal });
            }}
            onDelete={() => {
              const next = { ...mapObj };
              delete next[entryKey];
              setVal(next);
            }}
          />
        {/each}
      </div>
    {/if}
  </div>
{:else if schemaType === "array"}
  {@const arr = arrayValue}
  {@const itemSchema = (schema.items || { type: "string" }) as JsonSchemaNode}
  <div class="rounded-lg border border-[var(--border)] bg-[var(--surface-elevated)]/20 p-3.5 space-y-2.5">
    <div class="flex items-center justify-between gap-2">
      <div>
        <div class="text-xs font-semibold">{label} ({arr.length})</div>
        {#if schema.description}
          <p class="text-[11px] text-[var(--muted)]">{schema.description}</p>
        {/if}
      </div>
      {#if !readonly}
        <button
          type="button"
          onclick={() => {
            setVal([...arr, defaultForSchema(itemSchema)]);
          }}
          class="inline-flex items-center gap-1 px-2.5 py-1 rounded border border-[var(--border)] bg-[var(--surface)] hover:border-[var(--accent)] text-xs cursor-pointer"
        >
          <Plus class="w-3.5 h-3.5" />
          <span>Add</span>
        </button>
      {/if}
    </div>
    {#if arr.length === 0}
      <div class="text-xs text-[var(--muted)]">No items.</div>
    {:else}
      <div class="space-y-2">
        {#each arr as item, idx (idx)}
          <ConfigSchemaField
            fieldKey={`${fieldKey} #${idx + 1}`}
            value={item}
            schema={itemSchema}
            {accounts}
            depth={depth + 1}
            {readonly}
            onValueChange={(nextVal) => {
              setVal(arr.map((v, i) => (i === idx ? nextVal : v)));
            }}
            onDelete={() => {
              setVal(arr.filter((__, i) => i !== idx));
            }}
          />
        {/each}
      </div>
    {/if}
  </div>
{:else}
  <!-- Primitive field: string / number / integer / enum -->
  <div class="space-y-1">
    <div class="flex items-center justify-between gap-2">
      <label class="text-xs font-medium text-[var(--text)]" for={`cfg-in-${fieldKey}-${depth}`}>
        {label}{#if required}<span class="text-[var(--loss)] ml-0.5">*</span>{/if}
      </label>
      {#if onDelete && !readonly}
        <button
          type="button"
          onclick={onDelete}
          title="Remove"
          class="p-0.5 text-[var(--muted)] hover:text-[var(--loss)] cursor-pointer"
        >
          <Trash2 class="w-3 h-3" />
        </button>
      {/if}
    </div>

    {#if schema.enum && Array.isArray(schema.enum)}
      <select
        id={`cfg-in-${fieldKey}-${depth}`}
        disabled={readonly}
        value={value ?? schema.default ?? ""}
        onchange={(e) => setVal((e.currentTarget as HTMLSelectElement).value)}
        class="w-full px-3 py-1.5 rounded-lg bg-[var(--surface)] border border-[var(--border)] text-xs outline-none focus:border-[var(--accent)]"
      >
        {#each schema.enum as opt}
          <option value={String(opt)}>{String(opt)}</option>
        {/each}
      </select>
    {:else if schemaType === "number" || schemaType === "integer"}
      <input
        id={`cfg-in-${fieldKey}-${depth}`}
        type="number"
        step={schemaType === "integer" ? "1" : "any"}
        disabled={readonly}
        value={value ?? schema.default ?? ""}
        oninput={(e) => {
          const raw = (e.currentTarget as HTMLInputElement).value;
          setVal(raw === "" ? undefined : Number(raw));
        }}
        class="w-full px-3 py-1.5 rounded-lg bg-[var(--surface)] border border-[var(--border)] text-xs outline-none focus:border-[var(--accent)]"
      />
    {:else}
      <input
        id={`cfg-in-${fieldKey}-${depth}`}
        type={schema["ui:widget"] === "password" ? "password" : "text"}
        list={schema["ui:widget"] === "account" || fieldKey.includes("account")
          ? "cfg-accounts-datalist"
          : undefined}
        disabled={readonly}
        value={value ?? schema.default ?? ""}
        oninput={(e) => setVal((e.currentTarget as HTMLInputElement).value)}
        class="w-full px-3 py-1.5 rounded-lg bg-[var(--surface)] border border-[var(--border)] text-xs font-mono outline-none focus:border-[var(--accent)]"
      />
    {/if}

    {#if schema.description}
      <p class="text-[11px] text-[var(--muted)]">{schema.description}</p>
    {/if}
  </div>
{/if}

{#if depth === 0}
  <datalist id="cfg-accounts-datalist">
    {#each accounts as acc}
      <option value={acc}></option>
    {/each}
  </datalist>
{/if}
