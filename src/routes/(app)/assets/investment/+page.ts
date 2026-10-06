import { ajax } from "$lib/utils";
import type { PageLoad } from "./$types";

export const load = (async () => {
  const [investment, currencies] = await Promise.all([
    ajax("/api/investment"),
    ajax("/api/price/currencies")
  ]);

  return {
    investment,
    currencies
  };
}) satisfies PageLoad;
