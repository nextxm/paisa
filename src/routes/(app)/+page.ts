import { ajax } from "$lib/utils";
import type { PageLoad } from "./$types";

export const load = (async () => {
  const [dashboard, income] = await Promise.all([
    ajax("/api/dashboard"),
    ajax("/api/income/investment")
  ]);

  return {
    dashboard,
    income
  };
}) satisfies PageLoad;
