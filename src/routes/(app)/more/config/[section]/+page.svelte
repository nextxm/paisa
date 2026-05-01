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
