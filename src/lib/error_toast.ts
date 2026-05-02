function formatError(error: any) {
  if (error == null) {
    return "Unknown error";
  }

  if (typeof error === "string") {
    return error;
  }

  if (error.stack) {
    return error.stack;
  }

  if (error.message) {
    return error.message;
  }

  return String(error);
}

function escapeHtml(text: string) {
  return text
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/\"/g, "&quot;")
    .replace(/'/g, "&#39;");
}

const footer = `
<p class="mt-3">
  Please report this issue at <a href="https://github.com/nextxm/paisa/issues"
    >https://github.com/nextxm/paisa/issues</a
  >. Closing and reopening the app may help.
</p>
`;

export function buildErrorToastMessage(error: any) {
  const message = escapeHtml(formatError(error));
  return `<article class="notification is-danger is-light invertable"><p class="has-text-weight-semibold mb-2">Something Went Wrong</p><pre style="white-space: pre-wrap; max-height: 30vh; overflow: auto; margin: 0;">${message}</pre>${footer}</article>`;
}
