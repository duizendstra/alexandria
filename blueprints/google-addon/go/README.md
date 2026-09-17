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

```bash
# Set your target GCP registry
export KO_DOCKER_REPO=europe-west1-docker.pkg.dev/YOUR_PROJECT_ID/services

# Build and deploy directly to Cloud Run
gcloud run deploy calendar-time-tracker \
  --image=$(ko build .) \
  --region=europe-west1 \
  --allow-unauthenticated \
  --set-env-vars="GEMINI_API_KEY=your-gemini-api-key"
```

## Connecting to Google Workspace Add-on Manifest

In your Google Cloud project (or Google Workspace Add-on deployment), configure the HTTP endpoint URL pointing to your Cloud Run service URL (e.g., `https://calendar-time-tracker-xxxx-ew.a.run.app`).
