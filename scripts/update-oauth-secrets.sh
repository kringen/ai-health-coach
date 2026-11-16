#!/bin/bash
# Script to update Google OAuth2 credentials
# Usage: ./update-oauth-secrets.sh YOUR_CLIENT_ID YOUR_CLIENT_SECRET

CLIENT_ID="$1"
CLIENT_SECRET="$2"
NAMESPACE="health-coach-development"

if [ -z "$CLIENT_ID" ] || [ -z "$CLIENT_SECRET" ]; then
    echo "Usage: $0 <google-client-id> <google-client-secret>"
    echo "Example: $0 123456789-abcdef.apps.googleusercontent.com your-secret-key"
    exit 1
fi

echo "Updating Google OAuth2 credentials in Kubernetes..."
echo "Namespace: $NAMESPACE"

# Find the actual secret name that the deployment is using
SECRET_NAME=$(kubectl get deployment health-coach-dev -n "$NAMESPACE" -o jsonpath='{.spec.template.spec.containers[0].env[?(@.name=="GOOGLE_CLIENT_ID")].valueFrom.secretKeyRef.name}')

if [ -z "$SECRET_NAME" ]; then
    echo "❌ Error: Could not find the secret name used by the deployment"
    echo "Available secrets:"
    kubectl get secrets -n "$NAMESPACE"
    exit 1
fi

echo "Found deployment secret: $SECRET_NAME"

# Check if manually created secret exists with real credentials
MANUAL_SECRET="app-secrets-dev"
if kubectl get secret "$MANUAL_SECRET" -n "$NAMESPACE" >/dev/null 2>&1; then
    echo "Found manual secret: $MANUAL_SECRET"
    echo "Copying credentials from manual secret to deployment secret..."
    
    # Get credentials from manual secret
    EXISTING_CLIENT_ID=$(kubectl get secret "$MANUAL_SECRET" -n "$NAMESPACE" -o jsonpath='{.data.google-client-id}')
    EXISTING_CLIENT_SECRET=$(kubectl get secret "$MANUAL_SECRET" -n "$NAMESPACE" -o jsonpath='{.data.google-client-secret}')
    
    if [ -n "$EXISTING_CLIENT_ID" ] && [ -n "$EXISTING_CLIENT_SECRET" ]; then
        echo "Using existing credentials from $MANUAL_SECRET"
        kubectl patch secret "$SECRET_NAME" -n "$NAMESPACE" --type='json' -p="[
          {\"op\": \"replace\", \"path\": \"/data/google-client-id\", \"value\": \"$EXISTING_CLIENT_ID\"},
          {\"op\": \"replace\", \"path\": \"/data/google-client-secret\", \"value\": \"$EXISTING_CLIENT_SECRET\"}
        ]"
    else
        echo "Manual secret exists but credentials are empty, using provided credentials"
        kubectl patch secret "$SECRET_NAME" -n "$NAMESPACE" --type='json' -p="[
          {\"op\": \"replace\", \"path\": \"/data/google-client-id\", \"value\": \"$(echo -n "$CLIENT_ID" | base64)\"},
          {\"op\": \"replace\", \"path\": \"/data/google-client-secret\", \"value\": \"$(echo -n "$CLIENT_SECRET" | base64)\"}
        ]"
    fi
else
    echo "No manual secret found, using provided credentials"
    kubectl patch secret "$SECRET_NAME" -n "$NAMESPACE" --type='json' -p="[
      {\"op\": \"replace\", \"path\": \"/data/google-client-id\", \"value\": \"$(echo -n "$CLIENT_ID" | base64)\"},
      {\"op\": \"replace\", \"path\": \"/data/google-client-secret\", \"value\": \"$(echo -n "$CLIENT_SECRET" | base64)\"}
    ]"
fi

echo "Restarting deployment to pick up new credentials..."
kubectl rollout restart deployment/health-coach-dev -n "$NAMESPACE"

echo "Waiting for deployment to be ready..."
kubectl rollout status deployment/health-coach-dev -n "$NAMESPACE"

echo "✅ Google OAuth2 credentials updated successfully!"
echo "You can now log in at: https://tracker.kringen.io"
