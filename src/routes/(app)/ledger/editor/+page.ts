import { redirect } from "@sveltejs/kit";
import { ajax, appPath } from "$lib/utils";
import type { PageLoad } from "./$types";

export const load: PageLoad = async () => {
  const { files } = await ajax("/api/editor/files");
  redirect(307, appPath(`/ledger/editor/${encodeURIComponent(files[0].name)}`));
};
