import { ajax } from "./utils";
import { jobs } from "./stores/jobs";

export async function sync(request: Record<string, any>): Promise<string> {
  const { job_id } = await ajax("/api/sync", {
    method: "POST",
    body: JSON.stringify(request)
  });

  jobs.upsert({
    id: job_id,
    status: "pending",
    created_at: new Date().toISOString()
  });

  return job_id;
}
