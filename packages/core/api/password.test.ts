// @vitest-environment node
import { afterEach, describe, expect, it, vi } from "vitest";
import { ApiClient } from "./client";
import { AppConfigSchema } from "./schemas";

afterEach(() => vi.unstubAllGlobals());

describe("password authentication boundary", () => {
  it("keeps password login disabled on older or malformed server configurations", () => {
    expect(AppConfigSchema.parse({}).password_auth_enabled).toBe(false);
    expect(AppConfigSchema.parse({ password_auth_enabled: "true" }).password_auth_enabled).toBe(false);
  });

  it.each([{ token: "", user: { id: "user-1" } }, { token: "token", user: null }, {}])(
    "rejects malformed successful login responses: %j", async (body) => {
      vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(JSON.stringify(body))));
      const client = new ApiClient("http://localhost:3000");
      await expect(client.passwordLogin("owner@example.com", "test-password")).rejects.toThrow("Invalid password login response");
    },
  );

  it("rejects malformed password status instead of offering first-time setup", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(JSON.stringify({ configured: "false" }))));
    await expect(new ApiClient("http://localhost:3000").getPasswordStatus()).rejects.toThrow("Invalid password status response");
  });
});
