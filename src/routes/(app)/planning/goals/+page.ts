import { fetchConfig } from "$lib/config_client";
import { ajax } from "$lib/utils";
import type { PageLoad } from "./$types";

export const load = (async () => {
  const [{ config }, { goals }] = await Promise.all([fetchConfig(), ajax("/api/goals")]);
  config.goals = {
    retirement: config.goals?.retirement || [],
    savings: config.goals?.savings || [],
    life: config.goals?.life || []
  };
  return { config, goals };
}) satisfies PageLoad;
