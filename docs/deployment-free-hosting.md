# Deploying Paisa to Free Hosting

This guide covers deploying Paisa to two excellent free hosting platforms: **Render.com** (easiest) and **Oracle Cloud Free Tier** (most generous).

## Table of Contents

- [Render.com Deployment](#rendercom-deployment)
- [Oracle Cloud Free Tier Deployment](#oracle-cloud-free-tier-deployment)
- [Comparison](#comparison)

---

## Render.com Deployment

**Time to deploy:** ~5-10 minutes  
**Cost:** Free (750 free dyno-hours/month = ~24/7 uptime)  
**Requirements:** GitHub account

### Step 1: Prepare Your Repository

Ensure your GitHub repository is up to date with all changes:

```bash
git add .
git commit -m "Ready for Render deployment"
git push origin main
```

### Step 2: Create a Render Account

1. Go to [render.com](https://render.com)
2. Click **"Sign up"**
3. Choose **"Sign up with GitHub"** (easiest)
4. Authorize Render to access your GitHub account
5. Accept the terms and complete signup

### Step 3: Create a New Web Service

1. In Render dashboard, click **"New +"** button (top right)
2. Select **"Web Service"**
3. Choose your **paisa** repository
4. Click **"Connect"**

### Step 4: Configure Deployment

Fill in the deployment settings:

| Setting | Value |
|---------|-------|
| **Name** | `paisa-demo` (or your preferred name) |
| **Environment** | `Docker` |
| **Region** | Choose closest to you (e.g., `US East`) |
| **Branch** | `main` |
| **Auto-Deploy** | Toggle ON (auto-deploy on git push) |

### Step 5: Set Environment Variables

Click **"Advanced"** to add environment variables:

```
PAISA_CONFIG=/root/paisa.yaml
```

### Step 6: Configure Demo Data (Optional)

Add a demo `paisa.yaml` with read-only settings:

```yaml
server:
  port: 7500

ledger_cli: ledger

journals:
  - name: demo
    path: /root/demo.ledger

add_journal_path: null

default_currency: USD
```

Commit this to your repo at `demo-paisa.yaml`.

### Step 7: Deploy

1. Click **"Create Web Service"**
2. Render will start building (watch the logs)
3. Once deployed, you'll get a URL like: `https://paisa-demo.onrender.com`
4. Click the URL to view your live demo!

### Step 8: Auto-Deployment Setup

To auto-deploy on every GitHub push:

1. Render will show a webhook URL
2. Go to GitHub: Settings → Webhooks
3. Verify Render's webhook is already added (usually automatic)
4. Test by pushing a change: `git commit --allow-empty -m "test" && git push`

### Troubleshooting Render

| Issue | Solution |
|-------|----------|
| Build fails | Check logs in Render dashboard → Logs tab |
| App crashes | Verify `paisa.yaml` path is correct |
| Port issues | Render maps port 7500 to 443 automatically |
| Data resets | Render containers don't persist between deploys; use external DB for persistence |

---

## Oracle Cloud Free Tier Deployment

**Time to deploy:** ~30-45 minutes  
**Cost:** Free (actually free, forever)  
**Requirements:** GitHub account, credit card (won't be charged)  
**Persistence:** Full (data survives)

### Prerequisites

- Oracle Cloud account (free tier)
- SSH key pair generated
- Basic Linux familiarity

### Step 1: Set Up Oracle Cloud Account

1. Go to [oracle.com/cloud/free](https://www.oracle.com/cloud/free/)
2. Click **"Start for free"**
3. Complete registration (requires credit card, but won't be charged)
4. Verify email and phone
5. Log in to Oracle Cloud Console

### Step 2: Create an Instance

1. From Oracle Cloud Console, click **"Compute"** → **"Instances"**
2. Click **"Create Instance"**
3. Configure:
   - **Name:** `paisa-demo`
   - **Image:** `Ubuntu 22.04` (free tier eligible)
   - **Shape:** `Ampere (ARM64)` - Always Free Eligible
   - **OCPU:** 1 (free tier)
   - **Memory:** 6 GB (free tier)
4. **Networking:**
   - Create new VCN (Virtual Cloud Network)
   - Create new subnet
5. **SSH Key:**
   - Download your SSH private key (save as `paisa-key.pem`)
   - Set permissions: `chmod 600 paisa-key.pem`
6. Click **"Create"** (wait 1-2 minutes)

### Step 3: Configure Firewall Rules

1. Go to **VCN** → **Security Lists**
2. Click the security list for your subnet
3. Add **Ingress Rules:**
   - **Source:** `0.0.0.0/0`
   - **Protocol:** `TCP`
   - **Destination Port:** `80, 443, 7500`

### Step 4: Connect to Your Instance

Get the public IP from the instance details, then SSH in:

```bash
ssh -i paisa-key.pem ubuntu@<PUBLIC_IP>
```

### Step 5: Install Docker and Docker Compose

```bash
# Update system
sudo apt update && sudo apt upgrade -y

# Install Docker
sudo apt install -y docker.io docker-compose

# Add ubuntu user to docker group
sudo usermod -aG docker ubuntu

# Test Docker
docker --version

# Logout and login again for docker group to take effect
exit
ssh -i paisa-key.pem ubuntu@<PUBLIC_IP>
```

### Step 6: Clone Paisa Repository

```bash
cd ~
git clone https://github.com/YOUR_USERNAME/paisa.git
cd paisa
```

### Step 7: Build and Run Paisa

**Option A: Using Docker Compose (Recommended)**

Create `docker-compose.yml` in the paisa directory:

```yaml
version: '3.8'

services:
  paisa:
    build: .
    ports:
      - "7500:7500"
    volumes:
      - ./paisa-data:/root
      - ./paisa.yaml:/root/paisa.yaml:ro
    environment:
      - PAISA_CONFIG=/root/paisa.yaml
    restart: always
```

Create a demo `paisa.yaml`:

```yaml
server:
  port: 7500

ledger_cli: ledger

journals:
  - name: demo
    path: /root/demo.ledger

add_journal_path: /root/transactions.ledger

default_currency: USD
```

Run it:

```bash
# Build the Docker image
docker-compose build

# Start the service
docker-compose up -d

# View logs
docker-compose logs -f
```

**Option B: Direct Docker**

```bash
docker build -t paisa:latest .
docker run -d \
  --name paisa-demo \
  -p 7500:7500 \
  -v $(pwd)/paisa-data:/root \
  -e PAISA_CONFIG=/root/paisa.yaml \
  --restart always \
  paisa:latest
```

### Step 8: Set Up Reverse Proxy (Optional but Recommended)

Install Nginx to run on port 80/443:

```bash
sudo apt install -y nginx certbot python3-certbot-nginx

# Create Nginx config
sudo tee /etc/nginx/sites-available/paisa > /dev/null <<EOF
server {
    listen 80;
    server_name _;

    location / {
        proxy_pass http://localhost:7500;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
    }
}
EOF

# Enable the site
sudo ln -s /etc/nginx/sites-available/paisa /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl restart nginx

# (Optional) Get free SSL certificate
sudo certbot --nginx -d your-domain.com
```

### Step 9: Verify Deployment

```bash
# Check if Paisa is running
curl http://localhost:7500

# Check Docker container
docker ps

# View logs
docker logs -f paisa-demo
```

Access your demo at: `http://<PUBLIC_IP>:7500`

### Step 10: Set Up Auto-Updates (Optional)

Create a script to pull latest code and restart:

```bash
# ~/update-paisa.sh
#!/bin/bash
cd ~/paisa
git pull origin main
docker-compose down
docker-compose build
docker-compose up -d
```

Schedule it with cron:

```bash
crontab -e
# Add: 0 3 * * * ~/update-paisa.sh (runs daily at 3 AM)
```

### Troubleshooting Oracle Cloud

| Issue | Solution |
|-------|----------|
| Can't connect to instance | Check security list ingress rules; whitelist your IP |
| Port 7500 not accessible | Verify firewall rules allow port 7500 |
| Docker build fails | SSH back in and check `docker logs paisa-demo` |
| Out of disk space | Use `docker system prune` or increase volume |
| Instance stops | Instance may have shut down; restart from console |

---

## Comparison

| Feature | Render.com | Oracle Cloud |
|---------|-----------|--------------|
| **Cost** | Free (750h/mo) | Free (forever) |
| **Setup Time** | 5 min | 30-45 min |
| **Persistence** | No (ephemeral) | Yes (persistent) |
| **Database** | Not included | Add manually |
| **Auto-deploy** | Yes (GitHub webhook) | No (manual/scripted) |
| **Uptime** | 99.9% | ~99.9% |
| **Best For** | Quick demo | Long-term hosting |
| **Maintenance** | None | Minimal (OS patches) |

---

## Quick Start Comparison

### Render (5 minutes)
```
1. Sign up with GitHub
2. Connect repo
3. Click deploy
4. Done!
```

### Oracle Cloud (30 minutes)
```
1. Create account & instance
2. SSH in
3. Install Docker
4. Clone repo
5. docker-compose up
6. Done!
```

---

## Next Steps

- **Monitor your deployment** - Set up logs/alerts
- **Add custom domain** - Render and Oracle Cloud both support custom domains
- **Enable authentication** - Add a shared demo password in `paisa.yaml`
- **Set up backups** - For Oracle Cloud, export ledger periodically
- **Share your demo** - Add the link to your README!

---

## Support

- **Render Help:** https://render.com/docs
- **Oracle Cloud Help:** https://docs.oracle.com/
- **Paisa Issues:** https://github.com/nextxm/paisa/issues
