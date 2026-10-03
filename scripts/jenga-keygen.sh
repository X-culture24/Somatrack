#!/usr/bin/env bash
# Generates the RSA key pair Jenga uses to verify request signatures.
# Upload secrets/jenga_public.pem in Jenga HQ (Settings → API Keys); the
# private key stays on the server and is read via JENGA_PRIVATE_KEY_PATH.
set -euo pipefail
dir="$(cd "$(dirname "$0")/.." && pwd)/secrets"
mkdir -p "$dir"
chmod 700 "$dir"
if [[ -f "$dir/jenga_private.pem" ]]; then
  echo "secrets/jenga_private.pem already exists; refusing to overwrite." >&2
  exit 1
fi
openssl genrsa -out "$dir/jenga_private.pem" 4096
chmod 600 "$dir/jenga_private.pem"
openssl rsa -in "$dir/jenga_private.pem" -outform PEM -pubout -out "$dir/jenga_public.pem"
echo "Upload this public key to Jenga HQ:"
cat "$dir/jenga_public.pem"
