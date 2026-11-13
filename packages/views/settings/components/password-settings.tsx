"use client";

import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api } from "@multica/core/api";
import { useAuthStore } from "@multica/core/auth";
import { Button } from "@multica/ui/components/ui/button";
import { Input } from "@multica/ui/components/ui/input";
import { Label } from "@multica/ui/components/ui/label";
import { toast } from "sonner";
import { useT } from "../../i18n";
import { SettingsCard, SettingsSection } from "./settings-layout";

export function PasswordSettings() {
  const { t } = useT("auth");
  const userID = useAuthStore((state) => state.user?.id);
  const queryClient = useQueryClient();
  const [password, setPassword] = useState("");
  const [confirmation, setConfirmation] = useState("");
  const [currentPassword, setCurrentPassword] = useState("");
  const [validationError, setValidationError] = useState("");
  const config = useQuery({ queryKey: ["app-config"], queryFn: () => api.getConfig() });
  const statusKey = ["account-password", userID];
  const status = useQuery({
    queryKey: statusKey,
    queryFn: () => api.getPasswordStatus(),
    enabled: config.data?.password_auth_enabled === true && !!userID,
    retry: false,
  });
  const save = useMutation({
    mutationFn: () => api.setPassword(password, currentPassword),
    onSuccess: async () => {
      setPassword("");
      setConfirmation("");
      setCurrentPassword("");
      await queryClient.invalidateQueries({ queryKey: statusKey });
      toast.success(t(($) => $.password.saved));
    },
  });

  if (config.data?.password_auth_enabled !== true) return null;

  return (
    <SettingsSection title={t(($) => $.password.section)}>
      <SettingsCard>
        {status.isError ? <p role="alert" className="p-4 text-body text-destructive">{t(($) => $.password.load_failed)}</p> : (
          <form className="space-y-4 p-4" onSubmit={(event) => {
            event.preventDefault();
            setValidationError("");
            if (password !== confirmation) {
              setValidationError(t(($) => $.password.mismatch));
              return;
            }
            save.mutate();
          }}>
            {status.data?.configured && <div className="space-y-2">
              <Label htmlFor="current-password">{t(($) => $.password.current)}</Label>
              <Input id="current-password" type="password" autoComplete="current-password" required
                value={currentPassword} onChange={(event) => setCurrentPassword(event.target.value)} disabled={save.isPending} />
            </div>}
            <div className="space-y-2">
              <Label htmlFor="new-password">{t(($) => $.password.new_password)}</Label>
              <Input id="new-password" type="password" autoComplete="new-password" minLength={8} maxLength={72} required
                value={password} onChange={(event) => setPassword(event.target.value)} disabled={save.isPending} />
            </div>
            <div className="space-y-2">
              <Label htmlFor="confirm-password">{t(($) => $.password.confirm)}</Label>
              <Input id="confirm-password" type="password" autoComplete="new-password" required
                value={confirmation} onChange={(event) => setConfirmation(event.target.value)} disabled={save.isPending} />
            </div>
            <p className="text-body text-muted-foreground">{t(($) => $.password.requirements)}</p>
            {(validationError || save.error) && <p role="alert" className="text-body text-destructive">
              {validationError || save.error?.message}
            </p>}
            <Button type="submit" disabled={save.isPending || !status.data || !password || !confirmation}>
              {save.isPending ? t(($) => $.password.saving) : status.data?.configured ? t(($) => $.password.change) : t(($) => $.password.set)}
            </Button>
          </form>
        )}
      </SettingsCard>
    </SettingsSection>
  );
}
