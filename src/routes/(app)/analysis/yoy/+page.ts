import { redirect } from "@sveltejs/kit";
import type { PageLoad } from "./$types";

export const load = (() => {
  throw redirect(301, "/expense/yoy");
}) satisfies PageLoad;
