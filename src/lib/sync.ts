import * as toast from "bulma-toast";
import { ajax } from "./utils";
import { jobs } from "./stores/jobs";

export async function sync(request: Record<string, any>): Promise<string | null> {
  let job_id: string;
  try {
    ({ job_id } = await ajax("/api/sync", {
      method: "POST",
      body: JSON.stringify(request)
    }));
  } catch (err) {
    toast.toast({
      message: `<b>Failed to submit sync request</b>\n${
        err instanceof Error ? err.message : String(err)
      }`,
      type: "is-danger",
      duration: 10000
    });
    return null;
  }

  jobs.upsert({
    id: job_id,
    status: "pending",
    created_at: new Date().toISOString()
  });

  return job_id;
}
