import { fetchConfig } from "$lib/config_client";
import type { PageLoad } from "./$types";

export const load = (async () => {
  const { config } = await fetchConfig();
  config.goals = {
    retirement: config.goals?.retirement || [],
    savings: config.goals?.savings || [],
    life: config.goals?.life || []
  };
  return { config };
}) satisfies PageLoad;
