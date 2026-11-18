import dayjs from "dayjs";
import type { JsonObject } from "@bufbuild/protobuf";
import type { JSONSchema7 } from "json-schema";
import { loading } from "../store";
import { paisaClient } from "$lib/connect_client";

export type ConnectConfigResponse = {
  config: UserConfig;
  schema: JSONSchema7;
  now?: dayjs.Dayjs;
  accounts: string[];
  last_price_update: string;
  is_journal_dirty: boolean;
};

export type FetchConfigOptions = {
  background?: boolean;
  force?: boolean;
};

let cachedConfig: ConnectConfigResponse | null = null;
let inFlightConfig: Promise<ConnectConfigResponse> | null = null;

export function invalidateConfigCache(): void {
  cachedConfig = null;
}

export async function fetchConfig(opts?: FetchConfigOptions): Promise<ConnectConfigResponse> {
  if (!opts?.force && cachedConfig) {
    return cachedConfig;
  }

  if (!opts?.force && inFlightConfig) {
    return inFlightConfig;
  }

  if (!opts?.background) {
    loading.set(true);
  }

  const requestPromise = (async () => {
    const response = await paisaClient.getConfig({});
    const mapped: ConnectConfigResponse = {
      config: (response.config ?? {}) as unknown as UserConfig,
      schema: (response.schema ?? {}) as JSONSchema7,
      now: response.now ? dayjs(response.now) : undefined,
      accounts: response.accounts,
      last_price_update: response.lastPriceUpdate,
      is_journal_dirty: response.isJournalDirty
    };
    cachedConfig = mapped;
    return mapped;
  })();

  inFlightConfig = requestPromise;

  try {
    return await requestPromise;
  } finally {
    if (inFlightConfig === requestPromise) {
      inFlightConfig = null;
    }
    if (!opts?.background) {
      loading.set(false);
    }
  }
}

export async function updateConfig(
  config: UserConfig,
  opts?: { background?: boolean }
): Promise<{ success: boolean; error?: string }> {
  if (!opts?.background) {
    loading.set(true);
  }
  try {
    const response = await paisaClient.updateConfig({ config: config as unknown as JsonObject });
    if (response.success) {
      invalidateConfigCache();
    }
    return { success: response.success };
  } catch (e: any) {
    return { success: false, error: e?.message || "Failed to update config" };
  } finally {
    if (!opts?.background) {
      loading.set(false);
    }
  }
}
