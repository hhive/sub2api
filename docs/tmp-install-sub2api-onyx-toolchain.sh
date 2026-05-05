#!/usr/bin/env bash
set -euo pipefail

GO_VERSION="1.26.2"
GO_TARBALL="go${GO_VERSION}.linux-amd64.tar.gz"
GO_URL="https://go.dev/dl/${GO_TARBALL}"

if ! command -v /usr/local/go/bin/go >/dev/null 2>&1 || [ "$(/usr/local/go/bin/go version 2>/dev/null | awk '{print $3}')" != "go${GO_VERSION}" ]; then
  wget -O "/tmp/${GO_TARBALL}" "$GO_URL"
  rm -rf /usr/local/go
  tar -C /usr/local -xzf "/tmp/${GO_TARBALL}"
  cat >/etc/profile.d/go.sh <<'EOF'
export PATH=/usr/local/go/bin:$PATH
EOF
  chmod 644 /etc/profile.d/go.sh
fi

corepack enable
corepack prepare pnpm@10.33.2 --activate

if [ ! -x /root/.local/bin/uv ]; then
  curl -LsSf https://astral.sh/uv/install.sh -o /tmp/uv-install.sh
  sh /tmp/uv-install.sh
fi

/usr/local/go/bin/go version
pnpm -v
/root/.local/bin/uv --version
