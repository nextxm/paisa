---
description: "How to deploy Paisa securely for internet-facing access"
---

# Deployment

Paisa's built-in server listens on plain HTTP. For local-only use this is
perfectly fine, but **any internet-facing deployment must place Paisa behind a
reverse proxy that terminates TLS**. Without HTTPS an attacker on the network
can read your financial data and capture your session token or credentials.

!!! danger "HTTPS is required for internet-facing deployments"

    Never expose Paisa directly on a public IP or hostname without TLS.
    Always use a reverse proxy with a valid TLS certificate.

---

## How Paisa binds by default

By default `paisa serve` listens on `127.0.0.1:7500` (localhost only). The
reverse proxy runs on the public interface and forwards requests to Paisa on
localhost, so Paisa itself never needs to handle TLS.

```
Internet ──HTTPS──► Reverse Proxy (port 443) ──HTTP──► Paisa (127.0.0.1:7500)
```

---

## Nginx

The example below assumes you already have a TLS certificate. [Certbot with
Let's Encrypt](https://certbot.eff.org/) is the most common way to obtain a
free certificate automatically.

```nginx
server {
    listen 80;
    server_name paisa.example.com;

    # Redirect all plain-HTTP traffic to HTTPS.
    return 301 https://$host$request_uri;
}

server {
    listen 443 ssl;
    server_name paisa.example.com;

    ssl_certificate     /etc/letsencrypt/live/paisa.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/paisa.example.com/privkey.pem;

    # Modern TLS configuration (TLS 1.2+ only).
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers   HIGH:!aNULL:!MD5;

    # Optional: add HSTS to tell browsers to always use HTTPS.
    add_header Strict-Transport-Security "max-age=63072000" always;

    location / {
        proxy_pass         http://127.0.0.1:7500;
        proxy_http_version 1.1;
        proxy_set_header   Host              $host;
        proxy_set_header   X-Real-IP         $remote_addr;
        proxy_set_header   X-Forwarded-For   $proxy_add_x_forwarded_for;
        proxy_set_header   X-Forwarded-Proto $scheme;

        # Setting Connection to "" clears the hop-by-hop header forwarded by
        # Nginx, which is required to keep Server-Sent Event (SSE) streams
        # open. It is safe to apply globally to this location block.
        proxy_set_header   Connection        "";
        proxy_buffering    off;
    }
}
```

After updating the configuration, reload nginx:

```console
# sudo nginx -t && sudo systemctl reload nginx
```

---

## Caddy

[Caddy](https://caddyserver.com/) automatically provisions and renews Let's
Encrypt certificates. A minimal `Caddyfile` is all that is required:

```caddy
paisa.example.com {
    reverse_proxy 127.0.0.1:7500
}
```

Caddy handles HTTPS, HTTP→HTTPS redirects, and certificate renewal without any
extra configuration. Start or reload Caddy after saving the file:

```console
# caddy reload
```

---

## Docker with a reverse proxy

When running Paisa in Docker, bind the container port to localhost only so it
is not reachable directly from the internet, then front it with a reverse proxy
container.

### Docker Compose (Caddy + Paisa)

```yaml
services:
  paisa:
    image: ananthakumaran/paisa:latest
    volumes:
      - /home/john/Documents/paisa:/root/Documents/paisa
    # Bind to localhost only – do not publish 7500 to the host network.
    expose:
      - "7500"

  caddy:
    image: caddy:latest
    ports:
      - "80:80"
      - "443:443"
      - "443:443/udp"
    volumes:
      - ./Caddyfile:/etc/caddy/Caddyfile:ro
      - caddy_data:/data
      - caddy_config:/config

volumes:
  caddy_data:
  caddy_config:
```

`Caddyfile` (placed next to `docker-compose.yml`):

```caddy
paisa.example.com {
    reverse_proxy paisa:7500
}
```

Start the stack:

```console
# docker compose up -d
```

---

## Security checklist

Before exposing Paisa to the internet, verify the following:

- [ ] **HTTPS only** – Reverse proxy redirects HTTP → HTTPS (port 80 → 443).
- [ ] **User account configured** – At least one username/password has been
  added in the [User Authentication](./user-authentication.md) settings.
- [ ] **Strong password** – Use a randomly-generated password of 16+ characters.
- [ ] **Firewall** – Paisa's port (`7500`) is firewalled and only the reverse
  proxy can reach it.
- [ ] **HSTS** – `Strict-Transport-Security` header is set so browsers remember
  to use HTTPS.
- [ ] **Keep software updated** – Paisa, your reverse proxy, and the OS should
  all receive security updates regularly.
