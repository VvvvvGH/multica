# Password sign-in for self-hosted instances

Set `MULTICA_PASSWORD_AUTH=true` in the deployment environment and recreate the backend after applying migrations. The default is disabled, and older clients can continue using email codes or existing tokens.

Sign in once with an email verification code, then open **Settings → Profile → Password** to set a password. Afterward, use the email address and password on the login page. Changing an existing password requires the current password. Passwords must be 8–72 bytes and are stored as bcrypt hashes with cost 12 in the `user_password` table.

The new routes reuse the existing session cookies, CSRF protection, human-actor permission guard, and authentication rate limits. Password sign-in creates no accounts and accepts only an existing account with a configured password. Back up the database to preserve these credentials; keep `.env` and live data outside Git.

Use HTTPS when accessing the instance outside a trusted local network.

## Verification

Backend coverage exercises first-time setup, invalid lengths, wrong and unknown account credentials, session cookies, password changes, and the disabled feature flag. Frontend coverage exercises password submission, switching to email codes, confirmation matching, credential changes, and malformed API responses.

```sh
cd server
go test ./internal/handler -run '^TestPasswordAuthentication' -count=1
go test ./cmd/migrate ./internal/migrations
```

Handler tests require an explicit disposable `DATABASE_URL` with migrations already applied. They must not run against a deployed user database.

```sh
pnpm --filter @multica/core exec vitest run api/password.test.ts
pnpm --filter @multica/views exec vitest run auth/login-page.test.tsx settings/components/password-settings.test.tsx
```
