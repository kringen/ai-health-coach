# Kubernetes deployment

The manifests deploy the web app and connect it to an existing MariaDB server. They do not create or expose a database in the cluster.

## Configure MariaDB

Edit `base/kustomization.yaml` and replace `DB_HOST=mariadb.lan` with the MariaDB hostname or IP address reachable from the Kubernetes pods. The default port is `3306`; set the database name and user there as well. The MariaDB account needs permission to create and alter tables because the app runs its migrations during startup. The database must be MariaDB 10.6 or newer.

The Kubernetes nodes and pod network must be able to route to that LAN address, and the MariaDB firewall must allow connections from the cluster's source addresses. Do not use `localhost` or `127.0.0.1`; those refer to the app container. For a MariaDB server with a certificate trusted by the container's system roots, set `DB_TLS=true`; otherwise traffic uses the cluster's network without database TLS.

## Provision secrets and deploy

The Deployment wires the runtime settings used by the current Go code: environment/port, MariaDB connection, Google OAuth/session, and AI provider. The tracked base config selects Anthropic and retains Ollama settings for switching providers. Other `.env` entries such as metrics, rate limits, upload settings, Azure, and feature flags are not read by this application version.

Create the target namespace and the two Secrets before applying a Kustomize overlay. Put the MariaDB password in a local file named `mariadb-password`, and put these keys in local `app-secrets.env`: `google-client-id`, `google-client-secret`, `session-secret`, and `anthropic-api-key` when using Anthropic. Keep both files outside version control, or use your cluster's external secret manager integration:

```sh
# app-secrets.env contains the referenced app secret keys
kubectl create namespace health-coach-prod
kubectl create secret generic mariadb-secret \
  --namespace health-coach-prod \
  --from-file=password=./mariadb-password
kubectl create secret generic app-secrets \
  --namespace health-coach-prod \
  --from-env-file=./app-secrets.env
kubectl apply -k k8s/overlays/production
```

Use the namespace `health-coach-development` and matching Secret namespaces when deploying the development overlay. Do not commit these files.

Update the image name/tag in the selected overlay to a version built and pushed to a registry accessible by the cluster. The production ingress and `GOOGLE_REDIRECT_URL` currently both use `tracker.kringen.io`. If you change the public hostname, update both values and register the exact redirect URL in the Google OAuth client configuration.

The app uses the MariaDB migration set under `migrations/mariadb`. This creates a new MariaDB schema; it does not copy existing PostgreSQL data. Back up and migrate existing data separately before switching users over.