#!/bin/bash
# Script to update Google OAuth2 credentials
# Usage: ./update-oauth-secrets.sh YOUR_CLIENT_ID YOUR_CLIENT_SECRET

CLIENT_ID="$1"
CLIENT_SECRET="$2"

if [ -z "$CLIENT_ID" ] || [ -z "$CLIENT_SECRET" ]; then
    echo "Usage: $0 <google-client-id> <google-client-secret>"
    echo "Example: $0 123456789-abcdef.apps.googleusercontent.com your-secret-key"
    exit 1
fi

echo "Updating Google OAuth2 credentials in Kubernetes..."

kubectl patch secret app-secrets-dev -n health-coach-development --type='json' -p="[
  {\"op\": \"replace\", \"path\": \"/data/google-client-id\", \"value\": \"$(echo -n "$CLIENT_ID" | base64)\"},
  {\"op\": \"replace\", \"path\": \"/data/google-client-secret\", \"value\": \"$(echo -n "$CLIENT_SECRET" | base64)\"}
]"

echo "Restarting deployment to pick up new credentials..."
kubectl rollout restart deployment/health-coach-dev -n health-coach-development

echo "Waiting for deployment to be ready..."
kubectl rollout status deployment/health-coach-dev -n health-coach-development

echo "✅ Google OAuth2 credentials updated successfully!"
echo "You can now log in at: https://tracker.kringen.io"
