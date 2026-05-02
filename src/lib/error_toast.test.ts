import { describe, expect, test } from "bun:test";

import { buildErrorToastMessage } from "./error_toast";

describe("error toast", () => {
  test("does not include an inert inline delete button", () => {
    const message = buildErrorToastMessage(new Error("Boom"));

    expect(message).not.toContain('class="delete"');
    expect(message).toContain("Something Went Wrong");
  });

  test("escapes dangerous html in error output", () => {
    const message = buildErrorToastMessage('<script>alert("xss")</script>');

    expect(message).toContain("&lt;script&gt;alert(&quot;xss&quot;)&lt;/script&gt;");
    expect(message).not.toContain("<script>");
  });
});
