---
description: "How to secure Paisa by adding user account"
---

# User Authentication

You can setup username and password to make sure only you can access
the application. Go to `Configuration` page and expand the `User
Accounts` section. You can click the :fontawesome-solid-circle-plus:
icon to add a new username and password. Once you save the
configuration, you will get logged out if you had not logged in
earlier via another account.

It is important to understand that authentication only protects the
application user interface. If someone can access your computer and
they can access the folder where Paisa stores the ledger and database
files, they will be able to view your data.

!!! danger "HTTPS is required for internet-facing deployments"

    If you run Paisa on a server and access it over the public internet,
    you **must** use it over **HTTPS** and choose a **strong password**.
    Over plain HTTP, a [man-in-the-middle](https://en.wikipedia.org/wiki/Man-in-the-middle_attack)
    attacker can capture your session token and gain full access to your
    financial data. See the [Deployment](./deployment.md) guide for
    reverse-proxy TLS setup instructions.

## Session model

When you log in, Paisa creates a short-lived session and returns a
**session token** (a random UUID). This token is stored in the browser
and sent with every subsequent request in the `X-Auth` header.

| Property | Value |
|---|---|
| Token format | UUID v4 (random, 128-bit) |
| Token lifetime | 24 hours from login |
| Storage | SQLite `sessions` table on the server |
| Revocation | Logging out deletes the token immediately |

Session tokens expire automatically after 24 hours. Logging out (`POST
/api/auth/logout`) revokes the token immediately, regardless of the
remaining lifetime.

## Threat model

| Threat | Mitigation |
|---|---|
| Password guessing / brute-force | Login endpoint is rate-limited to 6 requests per minute per IP address |
| Token theft over the network | Tokens must only ever be transmitted over HTTPS (see warning above) |
| Token theft from config file | Tokens are **not** stored in `paisa.yaml`; only password hashes are |
| Session fixation / forgery | Tokens are cryptographically random UUIDs issued only after a successful login |
| Persistent access after logout | Logout deletes the token from the database immediately |
| Offline password cracking (stolen config file) | Passwords are hashed with SHA-256. While this prevents casual inspection, a determined attacker with access to `paisa.yaml` could attempt offline brute-force. Use a long, randomly-generated password and restrict file-system access to `paisa.yaml` to limit this risk. |

## Implementation details

### Password storage

Paisa uses the [SHA-256](https://en.wikipedia.org/wiki/Cryptographic_hash_function)
cryptographic hash function to convert your password into a digest before
storing it in the configuration file. This means:

- No one can read the configuration file and recover the password.
- If you forget the password you can remove the user accounts from the
  [configuration](./config.md) file to regain access.

Passwords are stored in `paisa.yaml` in the form `sha256:<hex-digest>`.

!!! warning "Use a strong, randomly-generated password"

    Paisa currently uses SHA-256 to hash passwords, which is a fast hash.
    If an attacker obtains a copy of `paisa.yaml`, they may be able to
    recover a weak password offline very quickly. Always use a long,
    randomly-generated password (16+ characters) and restrict read access
    to `paisa.yaml` on the server.

### Login flow

1. Browser sends `POST /api/auth/login` with `{"username": "…", "password": "…"}`.
2. Server hashes the supplied password with SHA-256 and compares it to the
   stored hash using a constant-time comparison (preventing timing attacks).
3. On success, a new session record is created in the database and the server
   returns `{"token": "<uuid>", "expires_at": "…", "username": "…"}`.
4. The browser stores the token and includes it as `X-Auth: <token>` on all
   subsequent requests.
5. Each request is validated against the `sessions` table; expired or
   non-existent tokens are rejected with HTTP 401.

### Legacy credential header

For backward compatibility, Paisa also supports a legacy authentication mode
where credentials are sent directly in every request as
`X-Auth: username:plaintext-password`. This mode is **disabled by default**
and should not be used for internet-facing deployments. It can be enabled in
`paisa.yaml` for automated scripts or API clients that cannot maintain sessions:

```yaml
allow_legacy_auth: true
```

Disable this option once all clients have migrated to the session-based login
flow.
