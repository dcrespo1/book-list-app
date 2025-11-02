#!/usr/bin/env bash
set -euo pipefail

echo "Running postCreate setup..."

# Make sure user-scoped Go paths exist and are writable
mkdir -p /home/vscode/.local/bin /home/vscode/go/pkg/mod
sudo chown -R vscode:vscode /home/vscode/go /home/vscode/.local/bin

# Initialize project
cd /workspaces/book-list-app

if [ ! -f .envrc ]; then
  echo "Creating .envrc from example..."
  cp .envrc.example .envrc
fi

# Allow direnv and initialize DB
if command -v direnv >/dev/null 2>&1; then
  echo "Running direnv setup..."
  direnv allow
  direnv exec . task init
else
  echo "⚠️ direnv not found — skipping direnv setup"
  task init
fi

echo "✅ postCreate completed successfully."
