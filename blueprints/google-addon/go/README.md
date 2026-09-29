# Google Calendar Time Tracker Add-on (Level 2: Go Alternate Runtime)

Enterprise-grade Google Workspace Alternate Runtime HTTP microservice written in Go and deployed to Google Cloud Run with zero Dockerfiles using `ko`.

## Features

- **Google Workspace Add-ons HTTP Webhook**: Parses incoming trigger events and returns Card JSON v2 responses.
- **Gemini Structured Output**: Uses Google AI Studio Gemini Flash with structured JSON schemas for high-speed extraction.
- **Alexandria SRE Stack**:
  - `go/platform/web`: Standardized HTTP JSON request/response handling.
  - `go/slog-gcp`: GCP Cloud Logging structured output and Cloud Trace context propagation.
- **Zero-Docker Deployment**: Pinned Chainguard static base image via `.ko.yaml`.

## Local Development

```bash
# Set your Gemini API key (from https://aistudio.google.com)
export GEMINI_API_KEY="your-gemini-api-key"

# Run locally
go run .
```

Test the HTTP trigger locally:
```bash
curl -X POST http://localhost:8080/ \
  -H "Content-Type: application/json" \
  -d '{"commonEventObject": {"hostApp": "CALENDAR"}}'
```

## Cloud Run Deployment with ko

Run these from `blueprints/google-addon/go`. The service is private: only the add-on's own service account may call it, and the Gemini key lives in Secret Manager, never on a command line or in the service's plain environment.

```bash
export PROJECT_ID=your-project-id
export REGION=europe-west1
gcloud config set project "$PROJECT_ID"

# 1. APIs: Cloud Run, Artifact Registry, Secret Manager, and Workspace add-ons.
gcloud services enable run.googleapis.com artifactregistry.googleapis.com \
  secretmanager.googleapis.com gsuiteaddons.googleapis.com

# 2. A registry for ko to push to, and Docker credentials for it.
gcloud artifacts repositories create services \
  --repository-format=docker --location="$REGION"
gcloud auth configure-docker "$REGION-docker.pkg.dev"
export KO_DOCKER_REPO="$REGION-docker.pkg.dev/$PROJECT_ID/services"

# 3. The Gemini key, into Secret Manager. read -s shows nothing as you paste.
read -rs GEMINI_API_KEY
printf %s "$GEMINI_API_KEY" | gcloud secrets create gemini-api-key --data-file=-
unset GEMINI_API_KEY

# 4. A runtime identity that can read that secret and nothing else.
gcloud iam service-accounts create time-tracker
export RUNTIME_SA="time-tracker@$PROJECT_ID.iam.gserviceaccount.com"
gcloud secrets add-iam-policy-binding gemini-api-key \
  --member="serviceAccount:$RUNTIME_SA" \
  --role=roles/secretmanager.secretAccessor

# 5. Build with ko and deploy. No unauthenticated access.
gcloud run deploy calendar-time-tracker \
  --image="$(ko build .)" \
  --region="$REGION" \
  --service-account="$RUNTIME_SA" \
  --set-secrets=GEMINI_API_KEY=gemini-api-key:latest \
  --no-allow-unauthenticated

# 6. Let the add-on, and only the add-on, invoke the service.
export ADDON_SA=$(gcloud workspace-add-ons get-authorization \
  --format='value(serviceAccountEmail)')
gcloud run services add-iam-policy-binding calendar-time-tracker \
  --region="$REGION" \
  --member="serviceAccount:$ADDON_SA" \
  --role=roles/run.invoker

# 7. Point an add-on deployment at the service and install it for yourself.
export SERVICE_URL=$(gcloud run services describe calendar-time-tracker \
  --region="$REGION" --format='value(status.url)')
sed "s#SERVICE_URL#$SERVICE_URL#" deployment.json > deployment.local.json
gcloud workspace-add-ons deployments create time-tracker \
  --deployment-file=deployment.local.json
gcloud workspace-add-ons deployments install time-tracker
```

Open Google Calendar and the add-on appears in the side panel. Each call carries an ID token for the add-on's service account, with the service URL as its audience; Cloud Run checks it before the request reaches Go. The service's button calls back the same URL, so every call passes the same check.

After a code change, re-run step 5 only; the URL, the grants and the add-on deployment stay as they are.
