import { ajax } from "$lib/utils";
import type { PageLoad } from "./$types";

export const load = (async () => {
  const dashboard = await ajax("/api/dashboard");

  return {
    dashboard
  };
}) satisfies PageLoad;
