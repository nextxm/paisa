<script lang="ts">
  import { getContext } from "svelte";
  import { page } from "$app/stores";
  import type { JSONSchema7 } from "json-schema";
  import JsonSchemaForm from "$lib/components/JsonSchemaForm.svelte";
  import _ from "lodash";
  import { ALL_SECTIONS, DEFAULT_SECTION_ID } from "$lib/config-sections";

  interface ConfigContext {
    config: any;
    schema: JSONSchema7 & { properties?: Record<string, any> };
    accounts: string[];
    error: string | null;
    isTogglingProviderDebug: boolean;
    applyProviderHTTPDebug: (enabled: boolean) => Promise<void>;
  }

  const ctx: ConfigContext = getContext("paisa-config");

  const sectionId = $derived($page.params.section ?? DEFAULT_SECTION_ID);
  const section = $derived(ALL_SECTIONS.find((s) => s.id === sectionId));

  let includeSecrets = $state(false);
  let isDownloadingBackup = $state(false);

  async function downloadBackup() {
    if (typeof window === "undefined" || isDownloadingBackup) return;
    isDownloadingBackup = true;
    try {
      const route = `/api/backup/export${includeSecrets ? "?include_secrets=true" : ""}`;
      const headers: Record<string, string> = {};
      const token = localStorage.getItem("token");
      if (token) {
        headers["X-Auth"] = token;
      }
      const response = await fetch(route, { headers });
      if (!response.ok) {
        throw new Error(`Failed to download backup: ${response.statusText}`);
      }

      const contentDisposition = response.headers.get("Content-Disposition");
      let filename = "paisa-backup.zip";
      if (contentDisposition) {
        const match = contentDisposition.match(/filename="?([^"]+)"?/);
        if (match && match[1]) {
          filename = match[1];
        }
      }

      const blob = await response.blob();
      const blobUrl = window.URL.createObjectURL(blob);
      const link = document.createElement("a");
      link.href = blobUrl;
      link.download = filename;
      document.body.appendChild(link);
      link.click();
      document.body.removeChild(link);
      window.URL.revokeObjectURL(blobUrl);
    } catch (err) {
      console.error("Backup download error:", err);
    } finally {
      isDownloadingBackup = false;
    }
  }
</script>

{#if section && ctx.config && ctx.schema?.properties}
  <div class="config-section-header">
    <h2 class="is-size-5 has-text-weight-semibold">
      <span class="icon-text">
        <span class="icon has-text-grey">
          <i class="fas {section.icon}"></i>
        </span>
        <span>{section.label}</span>
      </span>
    </h2>
    <p class="is-size-7 has-text-grey mt-1">{section.description}</p>
  </div>

  <div class="box config-section-box">
    <!-- Advanced section: provider HTTP debug live toggle (special UI) -->
    {#if sectionId === "advanced"}
      <article class="message is-warning is-small mb-4">
        <div class="message-body py-2 px-3">
          <div
            class="is-flex is-justify-content-space-between is-align-items-center is-flex-wrap-wrap gap-3"
          >
            <div>
              <b>Provider HTTP debug logging</b><br />
              <span class="is-size-7"
                >Toggle request/response logging immediately without saving the full form.</span
              >
            </div>
            <div class="field has-addons mb-0">
              <div class="control">
                <button
                  onclick={() => ctx.applyProviderHTTPDebug(false)}
                  class="button is-light is-small {ctx.isTogglingProviderDebug &&
                    !ctx.config.provider_debug_http &&
                    'is-loading'}"
                  disabled={ctx.isTogglingProviderDebug || !ctx.config.provider_debug_http}
                  >Disable</button
                >
              </div>
              <div class="control">
                <button
                  onclick={() => ctx.applyProviderHTTPDebug(true)}
                  class="button is-warning is-small {ctx.isTogglingProviderDebug &&
                    ctx.config.provider_debug_http &&
                    'is-loading'}"
                  disabled={ctx.isTogglingProviderDebug || ctx.config.provider_debug_http}
                  >Enable</button
                >
              </div>
            </div>
          </div>
        </div>
      </article>
    {/if}

    {#if sectionId === "backup"}
      <div class="backup-export-panel">
        <div class="message is-info is-small mb-4">
          <div class="message-body py-3 px-3">
            <h4 class="title is-6 mb-1">
              <i class="fas fa-file-zipper mr-1"></i> Full System Backup (.zip)
            </h4>
            <p class="is-size-7 mb-2">
              Generate a single archive containing your ledger journals, sheets, and configuration
              file.
            </p>
            <ul class="is-size-7 mb-0 ml-4" style="list-style-type: disc;">
              <li>
                Includes all ledger journal files (e.g. <code>main.ledger</code> and included files)
              </li>
              <li>Includes <code>paisa.yaml</code> configuration file</li>
              <li>Includes all sheet templates and data files</li>
              <li>Excludes SQLite cache DB (rebuilt automatically on sync)</li>
            </ul>
          </div>
        </div>

        <div class="box is-shadowless border p-4">
          <div class="field mb-4">
            <label class="checkbox is-size-7">
              <input type="checkbox" bind:checked={includeSecrets} />
              <span class="ml-1"
                >Include password hashes in <code>paisa.yaml</code> (default is sanitized)</span
              >
            </label>
          </div>

          <button
            type="button"
            class="button is-primary {isDownloadingBackup ? 'is-loading' : ''}"
            disabled={isDownloadingBackup}
            onclick={downloadBackup}
          >
            <span class="icon"><i class="fas fa-download"></i></span>
            <span>Download Backup (.zip)</span>
          </button>
        </div>
      </div>
    {/if}

    <!-- Render each schema key belonging to this section -->
    {#each section.schemaKeys as schemaKey}
      {#if ctx.schema.properties[schemaKey]}
        <JsonSchemaForm
          allAccounts={ctx.accounts}
          key={schemaKey}
          bind:value={ctx.config[schemaKey]}
          schema={ctx.schema.properties[schemaKey]}
          depth={0}
        />
      {/if}
    {/each}
  </div>
{:else if !ctx.config}
  <div class="has-text-centered py-6 has-text-grey">
    <span class="icon is-large">
      <i class="fas fa-spinner fa-spin"></i>
    </span>
  </div>
{:else}
  <div class="has-text-centered py-6 has-text-grey">
    <p>Section not found.</p>
  </div>
{/if}

<style lang="scss">
  .config-section-header {
    margin-bottom: 1rem;
  }

  .config-section-box {
    max-width: 900px;
  }
</style>
