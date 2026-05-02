import type { HandleClientError } from "@sveltejs/kit";
import dayjs from "dayjs";
import customParseFormat from "dayjs/plugin/customParseFormat";
dayjs.extend(customParseFormat);
import isSameOrBefore from "dayjs/plugin/isSameOrBefore";
dayjs.extend(isSameOrBefore);
import isSameOrAfter from "dayjs/plugin/isSameOrAfter";
dayjs.extend(isSameOrAfter);
import relativeTime from "dayjs/plugin/relativeTime";
dayjs.extend(relativeTime, {
  thresholds: [
    { l: "s", r: 1 },
    { l: "m", r: 1 },
    { l: "mm", r: 59, d: "minute" },
    { l: "h", r: 1 },
    { l: "hh", r: 23, d: "hour" },
    { l: "d", r: 1 },
    { l: "dd", r: 29, d: "day" },
    { l: "M", r: 1 },
    { l: "MM", r: 11, d: "month" },
    { l: "y", r: 1 },
    { l: "yy", d: "year" }
  ]
});
import utc from "dayjs/plugin/utc";
import timezone from "dayjs/plugin/timezone"; // dependent on utc plugin
dayjs.extend(utc);
dayjs.extend(timezone);
import localeData from "dayjs/plugin/localeData";
dayjs.extend(localeData);
import updateLocale from "dayjs/plugin/updateLocale";
dayjs.extend(updateLocale);

import * as pdfjs from "pdfjs-dist";
import pdfjsWorkerUrl from "pdfjs-dist/build/pdf.worker.mjs?url";

if (pdfjs.GlobalWorkerOptions) {
  pdfjs.GlobalWorkerOptions.workerSrc = pdfjsWorkerUrl;
}

import Handlebars from "handlebars";
import helpers from "$lib/template_helpers";
import * as toast from "bulma-toast";
import _ from "lodash";

import { buildErrorToastMessage } from "$lib/error_toast";

import "@formatjs/intl-numberformat/polyfill";
import "@formatjs/intl-numberformat/locale-data/en";

async function cleanupStaleLocalServiceWorker() {
  if (typeof window === "undefined") return;
  if (!("serviceWorker" in navigator)) return;

  const isLocalLikeHost =
    window.location.hostname === "localhost" ||
    window.location.hostname === "127.0.0.1" ||
    window.location.hostname === "phoenix" ||
    window.location.hostname.endsWith(".local");

  // During local/dev workflows, old SW caches can mix chunks from different builds.
  if (!(import.meta.env.DEV || isLocalLikeHost)) return;

  const regs = await navigator.serviceWorker.getRegistrations();
  if (regs.length === 0) return;

  await Promise.all(regs.map((reg) => reg.unregister()));

  if ("caches" in window) {
    const cacheKeys = await caches.keys();
    await Promise.all(cacheKeys.map((key) => caches.delete(key)));
  }

  if (!sessionStorage.getItem("paisa-sw-cleaned")) {
    sessionStorage.setItem("paisa-sw-cleaned", "1");
    window.location.reload();
  }
}

cleanupStaleLocalServiceWorker();

Handlebars.registerHelper(
  _.mapValues(helpers, (helper, name) => {
    return function (this: unknown, ...args: any[]) {
      try {
        return helper.apply(this, args);
      } catch (e) {
        console.log("Error in helper", name, args, e);
      }
    };
  })
);

toast.setDefaults({
  position: "bottom-right",
  dismissible: true,
  pauseOnHover: true,
  extraClasses: "is-light invertable"
});

globalThis.USER_CONFIG = {} as any;

export const handleError: HandleClientError = ({ error, status, message }) => {
  let stack: string | undefined;
  if (error instanceof Error) {
    stack = error.stack;
  }
  const detail = error == null ? "Unknown error" : String(error);
  return { message, stack, status, detail };
};

function displayError(error: any) {
  toast.toast({
    message: buildErrorToastMessage(error),
    type: "is-danger",
    dismissible: true,
    pauseOnHover: true,
    duration: 15000,
    position: "bottom-right",
    animate: { in: "fadeIn", out: "fadeOut" }
  });
}

window.addEventListener("unhandledrejection", (event) => {
  if (event.reason?.message?.includes("ResizeObserver loop limit exceeded")) {
    return;
  }
  displayError(event.reason);
});
window.addEventListener("error", (event) => {
  if (event.message?.includes("ResizeObserver loop limit exceeded")) {
    return;
  }
  displayError(event.error || event.message);
});
