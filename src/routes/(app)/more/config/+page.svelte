<script lang="ts">
  import { ajax, configUpdated } from "$lib/utils";
  import { onMount } from "svelte";
  import type { JSONSchema7 } from "json-schema";
  import JsonSchemaForm from "$lib/components/JsonSchemaForm.svelte";
  import _ from "lodash";
  import * as toast from "bulma-toast";
  import { refresh } from "../../../../store";
  import { sync } from "$lib/sync";

  let lastConfig: typeof globalThis.USER_CONFIG;
  let config: typeof globalThis.USER_CONFIG;
  let schema: JSONSchema7;
  let hasChanges = true;
  let isLoading = false;
  let isTogglingProviderDebug = false;
  let error: string = null;
  let accounts: string[] = [];
  onMount(async () => {
    ({ config, schema, accounts } = await ajax("/api/config"));
    lastConfig = _.cloneDeep(config);
  });

  async function resetToDefault() {
    if (
      confirm(
        "Are you sure you want to reset the config to defaults? This action is not reversible."
      )
    ) {
      save({
        journal_path: lastConfig.journal_path,
        db_path: lastConfig.db_path
      } as any);
    }
  }

  async function save(newConfig: typeof globalThis.USER_CONFIG) {
    isLoading = true;
    try {
      let success = false;
      ({ success, error } = await ajax("/api/config", {
        method: "POST",
        body: JSON.stringify(newConfig),
        background: true
      }));

      if (success) {
        lastConfig = _.cloneDeep(newConfig);
        config = _.cloneDeep(newConfig);
        globalThis.USER_CONFIG = _.cloneDeep(newConfig);
        configUpdated();
        refresh();
        toast.toast({
          message: `Saved config`,
          type: "is-success"
        });

        await sync({ journal: true });
      }
    } finally {
      isLoading = false;
    }
  }

  async function applyProviderHTTPDebug(enabled: boolean) {
    isTogglingProviderDebug = true;
    try {
      const response: {
        success?: boolean;
        enabled?: boolean;
        error?: { code: string; message: string };
      } = await ajax("/api/config/provider-debug-http", {
        method: "POST",
        body: JSON.stringify({ enabled }),
        background: true
      });

      if (response.success) {
        config.provider_debug_http = enabled;
        lastConfig.provider_debug_http = enabled;
        globalThis.USER_CONFIG = _.cloneDeep(config);
        configUpdated();
        toast.toast({
          message: `Provider HTTP debug logging ${enabled ? "enabled" : "disabled"}`,
          type: "is-success"
        });
        error = null;
      } else if (response.error?.message) {
        error = response.error.message;
      }
    } finally {
      isTogglingProviderDebug = false;
    }
  }

  $: hasChanges = !_.isEqual(config, lastConfig);
</script>

<div class="section">
  <div class="container is-fluid">
    <div class="columns">
      <div class="column is-12">
        {#if schema}
          <div class="box px-3" style="max-width: 1024px;">
            <article class="message">
              <div class="message-body">
                Prices are <b>not</b> automatically updated after config change. Use the menu at the
                top right corner to update prices. If the journal failed to sync due to any issues, fix
                the issues and use the menu to sync again.
              </div>
            </article>

            {#if error}
              <article class="message is-danger">
                <div class="message-body" style="overflow: auto; white-space: pre;">
                  {error}
                </div>
              </article>
            {/if}
            <article class="message is-warning">
              <div class="message-body">
                <div
                  class="is-flex is-justify-content-space-between is-align-items-center is-flex-wrap-wrap gap-3"
                >
                  <div>
                    <b>Provider HTTP debug logging</b><br />
                    Toggle request/response logging immediately for the next provider calls without saving
                    the full configuration form.
                  </div>
                  <div class="field has-addons mb-0">
                    <div class="control">
                      <button
                        on:click={() => applyProviderHTTPDebug(false)}
                        class="button is-light {isTogglingProviderDebug &&
                          !config.provider_debug_http &&
                          'is-loading'}"
                        disabled={isTogglingProviderDebug || !config.provider_debug_http}
                        >Disable</button
                      >
                    </div>
                    <div class="control">
                      <button
                        on:click={() => applyProviderHTTPDebug(true)}
                        class="button is-warning {isTogglingProviderDebug &&
                          config.provider_debug_http &&
                          'is-loading'}"
                        disabled={isTogglingProviderDebug || config.provider_debug_http}
                        >Enable</button
                      >
                    </div>
                  </div>
                </div>
              </div>
            </article>
            <div class="field is-grouped is-grouped-right">
              <div class="control">
                <button
                  on:click={() => save(config)}
                  class="button is-success {isLoading && 'is-loading'}"
                  disabled={!hasChanges}>Save</button
                >
              </div>
              <div class="control">
                <button on:click={() => (config = _.cloneDeep(lastConfig))} class="button is-light"
                  >Cancel</button
                >
              </div>
              <div class="control">
                <button on:click={() => resetToDefault()} class="button is-danger"
                  >Reset to Defaults</button
                >
              </div>
            </div>
            <JsonSchemaForm
              allAccounts={accounts}
              key="configuration"
              bind:value={config}
              {schema}
            />
          </div>
        {/if}
      </div>
    </div>
  </div>
</div>
