import { GlobalRegistrator } from "@happy-dom/global-registrator";
import { mock } from "bun:test";

GlobalRegistrator.register();

mock.module("$app/navigation", () => ({
  goto: async () => {},
  invalidateAll: async () => {},
  beforeNavigate: () => {},
  afterNavigate: () => {}
}));
