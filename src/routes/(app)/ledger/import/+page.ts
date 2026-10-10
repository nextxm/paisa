import { ajax } from "$lib/utils";
import type { PageLoad } from "./$types";

export const load = (async () => {
  const [accountTfIdf, templatesResult, presetsResult, rulesResult, accountsResult] =
    await Promise.all([
      ajax("/api/account/tf_idf"),
      ajax("/api/templates"),
      ajax("/api/import/presets"),
      ajax("/api/rules"),
      ajax("/api/accounts")
    ]);

  const templates = Array.isArray(templatesResult?.templates) ? templatesResult.templates : [];
  const importPresets = Array.isArray(presetsResult?.presets) ? presetsResult.presets : [];
  const rules = Array.isArray(rulesResult?.rules) ? rulesResult.rules : [];
  const accounts = Array.isArray(accountsResult?.accounts) ? accountsResult.accounts : [];

  return {
    accountTfIdf,
    templates,
    importPresets,
    rules,
    accounts
  };
}) satisfies PageLoad;
