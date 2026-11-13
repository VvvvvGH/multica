import type { ReactNode } from "react";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { I18nProvider } from "@multica/core/i18n/react";
import { api } from "@multica/core/api";
import enAuth from "../../locales/en/auth.json";
import { PasswordSettings } from "./password-settings";

vi.mock("@multica/core/api", () => ({ api: { getConfig: vi.fn(), getPasswordStatus: vi.fn(), setPassword: vi.fn() } }));
vi.mock("@multica/core/auth", () => ({ useAuthStore: (selector: (state: { user: { id: string } }) => unknown) => selector({ user: { id: "password-settings-user" } }) }));
vi.mock("./settings-layout", () => ({
  SettingsSection: ({ children, title }: { children: ReactNode; title: string }) => <section><h2>{title}</h2>{children}</section>,
  SettingsCard: ({ children }: { children: ReactNode }) => <div>{children}</div>,
}));

beforeEach(() => {
  vi.mocked(api.getConfig).mockResolvedValue({ cdn_domain: "", allow_signup: false, password_auth_enabled: true });
  vi.mocked(api.setPassword).mockResolvedValue();
});
afterEach(() => { cleanup(); vi.resetAllMocks(); });

function mount() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(<QueryClientProvider client={client}><I18nProvider locale="en" resources={{ en: { auth: enAuth } }}><PasswordSettings /></I18nProvider></QueryClientProvider>);
}

it("checks confirmation before saving the first password and reloads configured state", async () => {
  vi.mocked(api.getPasswordStatus).mockResolvedValueOnce({ enabled: true, configured: false }).mockResolvedValue({ enabled: true, configured: true });
  mount();
  const user = userEvent.setup();
  await user.type(await screen.findByLabelText("New password"), "test-password");
  await user.type(screen.getByLabelText("Confirm password"), "different-password");
  await user.click(screen.getByRole("button", { name: "Set password" }));
  expect(await screen.findByRole("alert")).toHaveTextContent("Passwords do not match");
  expect(api.setPassword).not.toHaveBeenCalled();
  await user.clear(screen.getByLabelText("Confirm password"));
  await user.type(screen.getByLabelText("Confirm password"), "test-password");
  await user.click(screen.getByRole("button", { name: "Set password" }));
  await waitFor(() => expect(api.setPassword).toHaveBeenCalledWith("test-password", ""));
  expect(await screen.findByLabelText("Current password")).toBeInTheDocument();
  expect(screen.getByLabelText("New password")).toHaveValue("");
});

it("requires the existing password when changing an established credential", async () => {
  vi.mocked(api.getPasswordStatus).mockResolvedValue({ enabled: true, configured: true });
  mount();
  const user = userEvent.setup();
  await user.type(await screen.findByLabelText("Current password"), "old-test-password");
  await user.type(screen.getByLabelText("New password"), "new-test-password");
  await user.type(screen.getByLabelText("Confirm password"), "new-test-password");
  await user.click(screen.getByRole("button", { name: "Change password" }));
  await waitFor(() => expect(api.setPassword).toHaveBeenCalledWith("new-test-password", "old-test-password"));
});
