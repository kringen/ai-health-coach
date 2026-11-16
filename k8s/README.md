# Kubernetes deployment

The manifests deploy the web app and connect it to an existing PostgreSQL server. They do not create or expose a database in the cluster.

## Configure PostgreSQL

Edit `base/kustomization.yaml` and replace `DB_HOST=postgres.lan` with the PostgreSQL hostname or IP address reachable from the Kubernetes pods. The default port is `5432`; set the database name, user, and `DB_SSLMODE` there as well. The PostgreSQL role needs permission to create and alter tables because the app runs its migrations during startup.

The Kubernetes nodes and pod network must be able to route to that LAN address, and PostgreSQL's firewall and `pg_hba.conf` must allow connections from the cluster's source addresses. Do not use `localhost` or `127.0.0.1`; those refer to the app container. Use an appropriate `DB_SSLMODE` value for your PostgreSQL server; `disable` is configured by default and should only be used on a trusted network.

## Provision secrets and deploy

The Deployment wires the runtime settings used by the current Go code: environment/port, PostgreSQL connection, Google OAuth/session, and AI provider. The tracked base config selects Anthropic and retains Ollama settings for switching providers. Other `.env` entries such as metrics, rate limits, upload settings, Azure, and feature flags are not read by this application version.

Create the target namespace and the two Secrets before applying a Kustomize overlay. Put the PostgreSQL password in `k8s/secrets/postgres-password`, and put these keys in `k8s/secrets/app-secrets.env`: `google-client-id`, `google-client-secret`, `session-secret`, and `anthropic-api-key` when using Anthropic. Keep both files outside version control, or use your cluster's external secret manager integration:

```sh
# app-secrets.env contains the referenced app secret keys
kubectl create namespace health-coach-prod
kubectl create secret generic postgres-secret \
  --namespace health-coach-prod \
  --from-file=password=./k8s/secrets/postgres-password
kubectl create secret generic app-secrets \
  --namespace health-coach-prod \
  --from-env-file=./app-secrets.env
kubectl apply -k k8s/overlays/production
```

Use the namespace `health-coach-development` and matching Secret namespaces when deploying the development overlay. Do not commit these files.

Update the image name/tag in the selected overlay to a version built and pushed to a registry accessible by the cluster. The production ingress and `GOOGLE_REDIRECT_URL` currently both use `tracker.kringen.io`. If you change the public hostname, update both values and register the exact redirect URL in the Google OAuth client configuration.

The app uses the PostgreSQL migration set under `migrations/`. The MariaDB driver and its separate migrations remain available but are not selected by Kubernetes.