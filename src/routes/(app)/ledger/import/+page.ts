import { ajax } from "$lib/utils";
import type { PageLoad } from "./$types";

export const load = (async () => {
  const [accountTfIdf, templatesResult, presetsResult] = await Promise.all([
    ajax("/api/account/tf_idf"),
    ajax("/api/templates"),
    ajax("/api/import/presets")
  ]);

  const templates = Array.isArray(templatesResult?.templates) ? templatesResult.templates : [];
  const importPresets = Array.isArray(presetsResult?.presets) ? presetsResult.presets : [];

  return {
    accountTfIdf,
    templates,
    importPresets
  };
}) satisfies PageLoad;
