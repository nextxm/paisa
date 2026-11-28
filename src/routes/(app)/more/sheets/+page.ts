import { redirect } from "@sveltejs/kit";
import { ajax, appPath } from "$lib/utils";
import type { PageLoad } from "./$types";

export const load: PageLoad = async () => {
  const { files } = await ajax("/api/sheets/files");
  if (files.length > 0) {
    redirect(307, appPath(`/more/sheets/${encodeURIComponent(files[0].name)}`));
  }
};
