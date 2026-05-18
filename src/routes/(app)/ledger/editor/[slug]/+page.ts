import { ajax } from "$lib/utils";
import type { PageLoad } from "./$types";

export const load = (async ({ params, url }) => {
  const lineNumber = Number(url.hash.substring(1));
  const { files, accounts, commodities, payees } = await ajax("/api/editor/files");

  return {
    name: params.slug,
    lineNumber: Number.isFinite(lineNumber) ? lineNumber : 0,
    files,
    accounts,
    commodities,
    payees
  };
}) satisfies PageLoad;
