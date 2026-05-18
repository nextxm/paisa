import { ajax } from "$lib/utils";
import type { PageLoad } from "./$types";

export const load = (async () => {
  const [accountTfIdf, templatesResult, presetsResult] = await Promise.all([
    ajax("/api/account/tf_idf"),
    ajax("/api/templates"),
    ajax("/api/import/presets")
  ]);

  return {
    accountTfIdf,
    templates: templatesResult.templates,
    importPresets: presetsResult.presets
  };
}) satisfies PageLoad;
