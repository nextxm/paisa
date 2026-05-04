<script lang="ts">
  import sha256 from "crypto-js/sha256";
  import type { JSONSchema7 } from "json-schema";
  import Select from "svelte-select";
  import _ from "lodash";
  import JsonSchemaForm from "./JsonSchemaForm.svelte";
  import PriceCodeSearchModal from "./PriceCodeSearchModal.svelte";
  import { iconGlyph, iconsList } from "$lib/icon";
  import AccountSelect from "./AccountsSelect.svelte";

  interface Schema extends JSONSchema7 {
    "ui:header"?: string;
    "ui:widget"?: string;
    "ui:order"?: number;
    "ui:open"?: boolean;
  }

  const ICON_MAX_RESULTS = 200;

  let {
    key,
    value = $bindable<any>(),
    rawValue = $bindable(""),
    schema,
    depth = 0,
    required = false,
    deletable = null,
    disabled = false,
    allAccounts,
    modalOpen = $bindable(false),
    compact = false
  }: {
    key: string;
    value: any;
    rawValue?: string;
    schema: Schema;
    depth?: number;
    required?: boolean;
    deletable?: () => void;
    disabled?: boolean;
    allAccounts: string[];
    modalOpen?: boolean;
    compact?: boolean;
  } = $props();

  let open = $state(false);
  let openInitialized = false;
  $effect(() => {
    if (!openInitialized && schema) {
      open = schema["ui:open"] ?? depth < 1;
      openInitialized = true;
    }
  });
  const title = $derived(_.startCase(key));

  function defaultValueForSchema(schema: any): any {
    if (!schema) return null;
    if (schema.default !== undefined) return _.cloneDeep(schema.default);

    if (schema.type === "string") return "";
    if (schema.type === "integer" || schema.type === "number") return 0;
    if (schema.type === "boolean") return false;
    if (schema.type === "array") return [];
    if (schema.type === "object") return {};

    return null;
  }

  function defaultCurrency(): string {
    return (globalThis as any)?.USER_CONFIG?.default_currency || "";
  }

  function newItem(schema: any, fieldKey: string, existingValues: any[] = []) {
    if (fieldKey === "currencies") {
      const candidate = defaultCurrency().trim();
      const alreadyExists = existingValues.some(
        (v) =>
          String(v || "")
            .trim()
            .toUpperCase() === candidate.toUpperCase()
      );
      return alreadyExists ? "" : candidate;
    }

    if (Array.isArray(schema?.default) && schema.default.length > 0) {
      return _.cloneDeep(schema.default[0]);
    }

    return defaultValueForSchema(schema?.items);
  }

  function sortedProperties(schema: Schema) {
    return _.sortBy(Object.entries(schema.properties), ([key, subSchema]: [string, Schema]) => {
      return [
        subSchema["ui:order"] || 999,
        _.includes(schema.required || [], key) ? 0 : 1,
        subSchema.type == "object" ? 2 : subSchema.type == "array" ? 3 : 1,
        key
      ];
    });
  }

  function documentation(schema: Schema) {
    if (schema.description) {
      return `<p style="max-width: 300px">${schema.description}</p>`;
    }
    return null;
  }

  async function searchIcons(text: string) {
    text = text.toLowerCase();
    if (_.isEmpty(text)) {
      return _.take(iconsList, ICON_MAX_RESULTS);
    }
    return _.take(
      iconsList.filter((icon) => icon.includes(text)),
      ICON_MAX_RESULTS
    );
  }
</script>

{#if deletable}
  <button
    type="button"
    onclick={(_e) => deletable()}
    class="config-delete"
    aria-label="Delete item"
  >
    <span class="icon is-small">
      <i class="fas fa-circle-minus"></i>
    </span>
  </button>
{/if}

{#if schema["ui:widget"] == "hidden"}
  <div></div>
{:else if schema["ui:widget"] == "password"}
  {#if compact}
    <div class="control">
      <input
        {disabled}
        {required}
        class="input is-small"
        type="password"
        bind:value={rawValue}
        onchange={() => {
          if (!_.isEmpty(rawValue)) {
            value = "sha256:" + sha256(sha256(rawValue).toString()).toString();
          }
        }}
      />
    </div>
  {:else}
    <div class="field is-horizontal">
      <div class="field-label is-small">
        <label data-tippy-content={documentation(schema)} for="" class="label">{title}</label>
      </div>
      <div class="field-body">
        <div class="field">
          <div class="control">
            <input
              {disabled}
              {required}
              class="input is-small"
              style="max-width: 350px;"
              type="password"
              bind:value={rawValue}
              onchange={() => {
                if (!_.isEmpty(rawValue)) {
                  value = "sha256:" + sha256(sha256(rawValue).toString()).toString();
                }
              }}
            />
          </div>
        </div>
      </div>
    </div>
  {/if}
{:else if schema["ui:widget"] == "icon"}
  {#if compact}
    <div class="control">
      <Select
        bind:justValue={value}
        class="icon-select is-small"
        {value}
        showChevron={true}
        loadOptions={searchIcons}
        searchable={true}
        clearable={!required}
      >
        <div class="custom-icon" slot="selection" let:selection>
          <span>{iconGlyph(selection.value)} {selection.value}</span>
        </div>
        <div class="custom-icon" slot="item" let:item>
          <span class="name">{iconGlyph(item.value)} {item.value}</span>
        </div>
      </Select>
    </div>
  {:else}
    <div class="field is-horizontal">
      <div class="field-label is-small">
        <label for="" data-tippy-content={documentation(schema)} class="label">{title}</label>
      </div>
      <div class="field-body">
        <div class="field">
          <div class="control" style="max-width: 350px">
            <Select
              bind:justValue={value}
              class="icon-select is-small"
              {value}
              showChevron={true}
              loadOptions={searchIcons}
              searchable={true}
              clearable={!required}
            >
              <div class="custom-icon" slot="selection" let:selection>
                <span>{iconGlyph(selection.value)} {selection.value}</span>
              </div>
              <div class="custom-icon" slot="item" let:item>
                <span class="name">{iconGlyph(item.value)} {item.value}</span>
              </div>
            </Select>
          </div>
        </div>
      </div>
    </div>
  {/if}
{:else if schema["ui:widget"] == "boolean"}
  {#if compact}
    <div class="control">
      <label class="radio">
        <input value="yes" bind:group={value} type="radio" />
        Yes
      </label>
      <label class="radio">
        <input value="no" bind:group={value} type="radio" />
        No
      </label>
    </div>
  {:else}
    <div class="field is-horizontal">
      <div class="field-label is-small">
        <label for="" data-tippy-content={documentation(schema)} class="label">{title}</label>
      </div>
      <div class="field-body">
        <div class="field">
          <div class="control">
            <label class="radio">
              <input value="yes" bind:group={value} type="radio" />
              Yes
            </label>
            <label class="radio">
              <input value="no" bind:group={value} type="radio" />
              No
            </label>
          </div>
        </div>
      </div>
    </div>
  {/if}
{:else if schema.type === "string" || _.isEqual(schema.type, ["string", "integer"])}
  {#if compact}
    <div class="control">
      {#if schema.enum}
        <div class="select is-small is-fullwidth">
          <select {disabled} bind:value {required}>
            {#each schema.enum as option}
              <option value={option}>{option}</option>
            {/each}
          </select>
        </div>
      {:else if schema["ui:widget"] == "textarea"}
        <textarea
          {disabled}
          {required}
          class="textarea is-small"
          rows="2"
          bind:value
          spellcheck="false"
          data-enable-grammarly="false"
        ></textarea>
      {:else}
        <input
          {disabled}
          {required}
          pattern={schema.pattern}
          class="input is-small"
          type="text"
          bind:value
        />
      {/if}
    </div>
  {:else}
    <div class="field is-horizontal">
      <div class="field-label is-small">
        <label data-tippy-content={documentation(schema)} for="" class="label">{title}</label>
      </div>
      <div class="field-body">
        <div class="field">
          <div class="control">
            {#if schema.enum}
              <div class="select is-small">
                <select {disabled} bind:value {required}>
                  {#each schema.enum as option}
                    <option value={option}>{option}</option>
                  {/each}
                </select>
              </div>
            {:else if schema["ui:widget"] == "textarea"}
              <textarea
                {disabled}
                {required}
                class="textarea is-small"
                style="min-width: 350px;max-width: 350px; width: 350px;"
                rows="5"
                bind:value
                spellcheck="false"
                data-enable-grammarly="false"
              ></textarea>
            {:else}
              <input
                {disabled}
                {required}
                pattern={schema.pattern}
                class="input is-small"
                style="max-width: 350px;"
                type="text"
                bind:value
              />
            {/if}
          </div>
        </div>
      </div>
    </div>
  {/if}
{:else if schema.type === "integer" || schema.type === "number"}
  {#if compact}
    <div class="control">
      <input
        {required}
        class="input is-small"
        type="number"
        min={schema.minimum}
        max={schema.maximum}
        step={schema.type == "integer" ? 1 : 0.01}
        bind:value
      />
    </div>
  {:else}
    <div class="field is-horizontal">
      <div class="field-label is-small">
        <label for="" data-tippy-content={documentation(schema)} class="label">{title}</label>
      </div>
      <div class="field-body">
        <div class="field">
          <div class="control">
            <input
              {required}
              class="input is-small"
              style="max-width: 350px;"
              type="number"
              min={schema.minimum}
              max={schema.maximum}
              step={schema.type == "integer" ? 1 : 0.01}
              bind:value
            />
          </div>
        </div>
      </div>
    </div>
  {/if}
{:else if schema["ui:widget"] == "accounts"}
  {#if compact}
    <div class="control">
      <AccountSelect {allAccounts} bind:accounts={value} />
    </div>
  {:else}
    <div class="field is-horizontal">
      <div class="field-label is-small">
        <label for="" data-tippy-content={documentation(schema)} class="label">{title}</label>
      </div>
      <div class="field-body">
        <div class="field">
          <div class="control pr-5">
            <AccountSelect {allAccounts} bind:accounts={value} />
          </div>
        </div>
      </div>
    </div>
  {/if}
{:else if schema["ui:widget"] == "price"}
  {#if compact}
    <div class="is-flex is-align-items-center gap-1">
      <div class="is-flex-grow-1">
        <div class="is-size-7 has-text-grey-darker">
          {value["code"] || "(empty)"}
        </div>
        <div class="is-size-7 has-text-grey">
          {value["provider"] || ""}
        </div>
      </div>
      <button
        type="button"
        onclick={(_e) => (modalOpen = true)}
        class="button is-small is-light"
        aria-label="Search price code"
      >
        <span class="icon is-small">
          <i class="fas fa-pen-to-square"></i>
        </span>
      </button>
    </div>
  {:else}
    <div class="config-header">
      <span class="is-link" data-tippy-content={documentation(schema)}>
        <span>{title}</span>
      </span>

      <button
        type="button"
        onclick={(_e) => (modalOpen = true)}
        class="button is-small"
        aria-label="Search price code"
      >
        <span class="icon is-small">
          <i class="fas fa-pen-to-square"></i>
        </span>
      </button>
    </div>
  {/if}

  <PriceCodeSearchModal
    bind:open={modalOpen}
    onselect={({ code, provider }) => {
      value = { ...value, code, provider };
    }}
  />

  {#if !compact}
    <div class="config-body {depth % 2 == 1 ? 'odd' : 'even'}">
      {#each sortedProperties(schema) as [key, subSchema]}
        <JsonSchemaForm
          {allAccounts}
          required={_.includes(schema.required || [], key)}
          depth={depth + 1}
          {key}
          bind:value={value[key]}
          schema={subSchema as Schema}
          disabled={true}
        />
      {/each}
    </div>
  {/if}
{:else if schema.type == "object"}
  {#if compact}
    <div class="is-flex is-flex-wrap-wrap gap-2">
      {#each sortedProperties(schema) as [key, subSchema]}
        <JsonSchemaForm
          {allAccounts}
          required={_.includes(schema.required || [], key)}
          depth={depth + 1}
          {key}
          bind:value={value[key]}
          schema={subSchema as Schema}
          compact={true}
        />
      {/each}
    </div>
  {:else}
    <div class="config-header">
      <button
        type="button"
        class="button is-light invertable is-small"
        data-tippy-content={documentation(schema)}
        onclick={(_e) => (open = !open)}
      >
        <span>{schema["ui:header"] ? value[schema["ui:header"]] || title : title}</span>
        <span class="icon is-small">
          <i class="fas {open ? 'fa-angle-up' : 'fa-angle-down'}"></i>
        </span>
      </button>
    </div>

    {#if open}
      <div class="config-body {depth % 2 == 1 ? 'odd' : 'even'}">
        {#each sortedProperties(schema) as [key, subSchema]}
          <JsonSchemaForm
            {allAccounts}
            required={_.includes(schema.required || [], key)}
            depth={depth + 1}
            {key}
            bind:value={value[key]}
            schema={subSchema as Schema}
          />
        {/each}
      </div>
    {/if}
  {/if}
{:else if schema.type === "boolean"}
  {#if compact}
    <label class="checkbox">
      <input type="checkbox" bind:checked={value} {disabled} />
    </label>
  {:else}
    <div class="field is-horizontal">
      <div class="field-label is-small">
        <label for="" data-tippy-content={documentation(schema)} class="label">{title}</label>
      </div>
      <div class="field-body">
        <div class="field">
          <div class="control">
            <label class="checkbox">
              <input type="checkbox" bind:checked={value} {disabled} />
            </label>
          </div>
        </div>
      </div>
    </div>
  {/if}
{:else if schema.type == "array"}
  {#if schema["ui:widget"] == "table"}
    <div class="table-container mb-4">
      <table class="table is-fullwidth config-table">
        <thead>
          <tr>
            {#each sortedProperties(schema.items as Schema) as [subKey, _subSchema]}
              <th class="is-size-7">{_.startCase(subKey)}</th>
            {/each}
            <th class="action-column"></th>
          </tr>
        </thead>
        <tbody>
          {#each Array.isArray(value) ? value : [] as _item, i}
            <tr>
              {#each sortedProperties(schema.items as Schema) as [subKey, subSchema]}
                <td>
                  <JsonSchemaForm
                    {allAccounts}
                    required={_.includes((schema.items as Schema).required || [], subKey)}
                    depth={depth + 1}
                    key={subKey}
                    bind:value={value[i][subKey]}
                    schema={subSchema as Schema}
                    compact={true}
                  />
                </td>
              {/each}
              <td class="action-column">
                <button
                  type="button"
                  onclick={() => {
                    if (Array.isArray(value)) {
                      value.splice(i, 1);
                      value = [...value];
                    }
                  }}
                  class="button is-small is-ghost has-text-danger"
                  aria-label="Delete item"
                >
                  <span class="icon is-small">
                    <i class="fas fa-trash-can"></i>
                  </span>
                </button>
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
      <button
        type="button"
        onclick={() => {
          const item = newItem(schema, key, Array.isArray(value) ? value : []);
          value = [item, ...(Array.isArray(value) ? value : [])];
        }}
        class="button is-small is-fullwidth is-dashed"
      >
        <span class="icon is-small">
          <i class="fas fa-plus"></i>
        </span>
        <span>Add {title}</span>
      </button>
    </div>
  {:else}
    <div class="config-header">
      <button
        type="button"
        class="button is-light invertable is-small"
        data-tippy-content={documentation(schema)}
        onclick={(_e) => (open = !open)}
      >
        <span>{title}</span>
        <span class="icon is-small">
          <i class="fas {open ? 'fa-angle-up' : 'fa-angle-down'}"></i>
        </span>
      </button>
      {#if open}
        <button
          type="button"
          onclick={(_e) =>
            (value = [
              newItem(schema, key, Array.isArray(value) ? value : []),
              ...(Array.isArray(value) ? value : [])
            ])}
          class="config-add"
          aria-label="Add item"
        >
          <span class="icon is-small">
            <i class="fas fa-circle-plus"></i>
          </span>
        </button>
      {/if}
    </div>

    {#if open}
      <div class="config-body {depth % 2 == 1 ? 'odd' : 'even'}">
        {#each Array.isArray(value) ? value : [] as _item, i}
          <JsonSchemaForm
            {allAccounts}
            deletable={() => {
              if (Array.isArray(value)) {
                value.splice(i, 1);
                value = [...value];
              }
            }}
            depth={depth + 1}
            key=""
            bind:value={value[i]}
            schema={schema.items as Schema}
          />
        {/each}
      </div>
    {/if}
  {/if}
{:else}
  <div>{JSON.stringify(schema)}</div>
{/if}
