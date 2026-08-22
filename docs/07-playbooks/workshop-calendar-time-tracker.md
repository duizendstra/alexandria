---
uuid: 7c90b350-89a8-4e8e-bc26-4163e950dcc2
title: "Workshop: Building an AI-Powered Google Calendar Time Tracker"
domain: "playbooks"
type: "guide"
diataxis_quadrant: "how-to"
status: "active"
maturity: "standard"
owner: "@duizendstra"
created_at: "2026-08-22T08:00:00Z"
updated_at: "2026-08-22T08:00:00Z"
summary: >
  Step-by-step hands-on workshop guide for building a Google Calendar time writing
  add-on powered by Gemini AI, progressing from a zero-setup Apps Script prototype
  to a production Go Alternate Runtime on Cloud Run.
audience: [public]
tags: [ "workshop", "calendar", "add-on", "apps-script", "go", "gemini-ai", "flossk" ]
relations: []
---

# Workshop: Building an AI-Powered Google Calendar Time Tracker

This guide provides the complete curriculum and technical runbook for a hands-on workshop (designed for FLOSSK and student developer communities) teaching how to build an intelligent Google Calendar Add-on for time tracking and time writing.

---

## Workshop Overview & Narrative

Tired of tedious timesheet forms? In this session, participants build an interactive Google Calendar sidebar that converts natural language work notes into structured, color-coded calendar events using Gemini AI.

The workshop follows a two-stage progression:
1. **Level 1 (Frictionless Prototyping):** Google Apps Script + `CardService` + `CalendarApp` + free Gemini Flash API key ($0 cost, 100% in-browser).
2. **Level 2 (Enterprise Scaling):** Go Alternate Runtime on Cloud Run + Alexandria SRE modules (`go/platform/web`, `go/slog-gcp`) + zero-Docker `ko` containerization.

```
+-------------------------------------------------------------------------------+
| LEVEL 1: Apps Script Prototype                                                |
| Google Calendar Sidebar  -->  Apps Script CardService  -->  Gemini API        |
|                                       │                                       |
|                                       v                                       |
|                             CalendarApp.createEvent()                         |
+-------------------------------------------------------------------------------+
                                        │
                         [ Graduate to Enterprise Scale ]
                                        │
                                        v
+-------------------------------------------------------------------------------+
| LEVEL 2: Enterprise Go Alternate Runtime                                      |
| Google Calendar Sidebar  -->  Cloud Run (Go + Alexandria)  -->  Gemini API    |
|                                       │                                       |
|                                       v                                       |
|                     Structured Card JSON v2 + SRE Tracing                     |
+-------------------------------------------------------------------------------+
```

---

## Prerequisites (Zero Cost)

1. **Google Account:** Any personal `@gmail.com` or Google Workspace account.
2. **Dedicated Calendar (Recommended):** In [Google Calendar](https://calendar.google.com), click **+** next to *Other calendars* > **Create new calendar** and name it `Time Tracker` (keeps time entries isolated from personal events).
3. **Free Gemini API Key:** Generated in 30 seconds with no credit card at [Google AI Studio](https://aistudio.google.com).
4. **Web Browser:** Google Chrome, Firefox, or Safari.

---

## Part 1: Level 1 — Apps Script Prototype (100% In-Browser)

### Step 1: Create the Apps Script Project

1. Navigate to [script.google.com](https://script.google.com).
2. Click **New project** and name it `Calendar Time Tracker AI`.

### Step 2: Configure the Manifest (`appsscript.json`)

1. Click **Project Settings** (gear icon on the left).
2. Check the box for **Show "appsscript.json" manifest file in editor**.
3. Return to the **Editor** (`< >`), open `appsscript.json`, and replace its contents with:

```json
{
  "timeZone": "Europe/Belgrade",
  "dependencies": {
    "enabledAdvancedServices": []
  },
  "exceptionLogging": "STACKDRIVER",
  "runtimeVersion": "V8",
  "oauthScopes": [
    "https://www.googleapis.com/auth/calendar.addons.execute",
    "https://www.googleapis.com/auth/calendar",
    "https://www.googleapis.com/auth/calendar.events",
    "https://www.googleapis.com/auth/script.external_request"
  ],
  "addOns": {
    "common": {
      "name": "Time Tracker AI",
      "logoUrl": "https://www.gstatic.com/images/icons/material/system/1x/schedule_black_24dp.png",
      "useLocaleFromApp": true
    },
    "calendar": {
      "homepageTrigger": {
        "runFunction": "onCalendarHomepage"
      }
    }
  }
}
```

### Step 3: Implement `Code.gs`

Open `Code.gs` and paste the implementation from the Alexandria blueprint:

```javascript
const SCRIPT_PROP_KEY = 'GEMINI_API_KEY';

function onCalendarHomepage(e) {
  const userProps = PropertiesService.getUserProperties();
  const apiKey = userProps.getProperty(SCRIPT_PROP_KEY);

  const card = CardService.newCardBuilder();
  card.setHeader(
    CardService.newCardHeader()
      .setTitle('AI Time Tracker')
      .setSubtitle('Log billable project time with Gemini AI')
  );

  const section = CardService.newCardSection();

  if (!apiKey) {
    section.addWidget(
      CardService.newTextParagraph().setText(
        'Step 1: Set your free Gemini API Key from aistudio.google.com'
      )
    );
    section.addWidget(
      CardService.newTextInput()
        .setFieldName('api_key')
        .setTitle('Gemini API Key')
    );
    section.addWidget(
      CardService.newTextButton()
        .setText('Save API Key')
        .setOnClickAction(CardService.newAction().setFunctionName('onSaveApiKey'))
        .setTextButtonStyle(CardService.TextButtonStyle.FILLED)
    );
    card.addSection(section);
    return card.build();
  }

  section.addWidget(
    CardService.newTextInput()
      .setFieldName('work_note')
      .setTitle('Work Note')
      .setMultiline(true)
      .setHint('e.g. Spent 1.5 hours debugging PostgreSQL query performance for Acme Corp')
  );

  section.addWidget(
    CardService.newTextButton()
      .setText('Analyze & Structure with AI')
      .setOnClickAction(CardService.newAction().setFunctionName('onAnalyzeNote'))
      .setTextButtonStyle(CardService.TextButtonStyle.FILLED)
  );

  card.addSection(section);
  return card.build();
}

function onSaveApiKey(e) {
  const apiKey = e.formInputs && e.formInputs.api_key ? e.formInputs.api_key[0].trim() : '';
  if (apiKey) {
    PropertiesService.getUserProperties().setProperty(SCRIPT_PROP_KEY, apiKey);
  }
  return CardService.newActionResponseBuilder()
    .setNavigation(CardService.newNavigation().updateCard(onCalendarHomepage(e)))
    .build();
}

function makeTimeEntry(client, project, hours, title, summary) {
  const safeClient = client ? String(client).trim() : 'General';
  const safeProject = project ? String(project).trim() : 'Work';
  const safeHours = typeof hours === 'number' && hours > 0 ? hours : 1.0;
  const safeTitle = title ? String(title).trim() : `[${safeClient}] ${safeProject}`;
  const safeSummary = summary ? String(summary).trim() : '';

  return Object.freeze({
    client: safeClient,
    project: safeProject,
    duration_hours: safeHours,
    title: safeTitle,
    summary: safeSummary
  });
}

function onAnalyzeNote(e) {
  const note = e.formInputs && e.formInputs.work_note ? e.formInputs.work_note[0].trim() : '';
  const apiKey = PropertiesService.getUserProperties().getProperty(SCRIPT_PROP_KEY);

  const url = 'https://generativelanguage.googleapis.com/v1beta/models/gemini-1.5-flash:generateContent?key=' + encodeURIComponent(apiKey);
  const prompt = `Extract time entry JSON from this note: "${note}".
Return ONLY raw JSON with: client (string), project (string), duration_hours (number), title (string), summary (string).`;

  const response = UrlFetchApp.fetch(url, {
    method: 'post',
    contentType: 'application/json',
    payload: JSON.stringify({
      contents: [{ parts: [{ text: prompt }] }],
      generationConfig: { responseMimeType: "application/json", temperature: 0.2 }
    })
  });

  const raw = JSON.parse(response.getContentText()).candidates[0].content.parts[0].text;
  const rawObj = JSON.parse(raw);
  const entry = makeTimeEntry(rawObj.client, rawObj.project, rawObj.duration_hours, rawObj.title, rawObj.summary);

  const confirmCard = CardService.newCardBuilder()
    .setHeader(CardService.newCardHeader().setTitle('Confirm Entry').setSubtitle(entry.title))
    .addSection(
      CardService.newCardSection()
        .addWidget(CardService.newTextInput().setFieldName('entry_title').setTitle('Title').setValue(entry.title))
        .addWidget(CardService.newTextInput().setFieldName('entry_duration').setTitle('Duration (Hours)').setValue(String(entry.duration_hours)))
        .addWidget(CardService.newTextInput().setFieldName('entry_summary').setTitle('Notes').setMultiline(true).setValue(entry.summary))
        .addWidget(
          CardService.newTextButton()
            .setText('Create Calendar Event')
            .setOnClickAction(CardService.newAction().setFunctionName('onLogToCalendar'))
            .setTextButtonStyle(CardService.TextButtonStyle.FILLED)
        )
    )
    .build();

  return CardService.newActionResponseBuilder()
    .setNavigation(CardService.newNavigation().pushCard(confirmCard))
    .build();
}

const CALENDAR_NAME = 'Time Tracker';

function getOrCreateTimeTrackerCalendar() {
  const cals = CalendarApp.getCalendarsByName(CALENDAR_NAME);
  if (cals.length > 0) {
    return cals[0];
  }
  return CalendarApp.createCalendar(CALENDAR_NAME, {
    summary: 'Dedicated calendar for AI Time Tracker entries'
  });
}

function onLogToCalendar(e) {
  const formInputs = e.formInputs || {};
  const title = formInputs.entry_title ? formInputs.entry_title[0] : 'Work Log';
  const hours = parseFloat(formInputs.entry_duration ? formInputs.entry_duration[0] : '1.0') || 1.0;
  const summary = formInputs.entry_summary ? formInputs.entry_summary[0] : '';

  const now = new Date();
  const start = new Date(now.getTime() - (hours * 3600 * 1000));
  const cal = getOrCreateTimeTrackerCalendar();
  const event = cal.createEvent(title, start, now, { description: summary });
  event.setColor(CalendarApp.EventColor.CYAN);

  const successCard = CardService.newCardBuilder()
    .setHeader(CardService.newCardHeader().setTitle('Time Logged!'))
    .addSection(
      CardService.newCardSection()
        .addWidget(CardService.newTextParagraph().setText(`Created event <b>${title}</b> (${hours}h) in <b>${CALENDAR_NAME}</b> calendar.`))
        .addWidget(CardService.newTextButton().setText('Log Another Entry').setOnClickAction(CardService.newAction().setFunctionName('onCalendarHomepage')))
    )
    .build();

  return CardService.newActionResponseBuilder()
    .setNavigation(CardService.newNavigation().popToRoot().updateCard(successCard))
    .build();
}
```

### Step 4: Test in Google Calendar

1. Click **Deploy** (top right) > **Test deployments**.
2. Click **Install**.
3. Open [Google Calendar](https://calendar.google.com).
4. On the right-side companion panel, click the **Time Tracker AI** icon.
5. Enter your Gemini API key, submit a test note (e.g. *"Spent 2 hours reviewing PRs for Acme"*), and watch the event appear inside your dedicated `Time Tracker` calendar!

---

## Part 2: The Architectural Tipping Point

While Apps Script is ideal for personal prototypes, engineering organizations face limitations:

| Dimension | Apps Script Limitation | Go Alternate Runtime Solution |
|---|---|---|
| **Execution Limits** | 6-minute hard execution timeout | Unlimited background processing / streaming |
| **Type Safety** | Untyped JavaScript | Strictly typed Go domain models with zero runtime type panics |
| **Observability** | Basic console logs | GCP Cloud Logging with trace propagation (`go/slog-gcp`) |
| **Resilience** | Ad-hoc error retries | Exponential backoff retries on rate limits (`go/retry`) |
| **Multi-Tenancy** | Single-user script properties | Secret Manager and enterprise database connectivity |

---

## Part 3: Level 2 — Go Alternate Runtime Microservice

Alexandria provides a pre-scaffolded blueprint in `blueprints/google-addon/go/`.

### Go Webhook Architecture

Google Workspace Alternate Runtimes POST JSON trigger events to an HTTP endpoint. The Go service processes the event, calls Gemini with a structured schema, and returns Card JSON v2.

```go
package main

import (
    "log/slog"
    "net/http"
    "os"
    _ "time/tzdata"

    sloggcp "github.com/duizendstra/alexandria/go/slog-gcp"
)

func main() {
    sloggcp.Setup()

    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }

    mux := http.NewServeMux()
    mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        _, _ = w.Write([]byte("ok"))
    })
    mux.HandleFunc("/", HandleCalendarTrigger)

    slog.Info("starting Google Calendar Add-on service", "port", port)
    if err := http.ListenAndServe(":"+port, sloggcp.TraceMiddleware(mux)); err != nil {
        slog.Error("server error", "err", err)
    }
}
```

### Local Testing

```bash
cd blueprints/google-addon/go
export GEMINI_API_KEY="your-gemini-key"
go run .
```

Send a test trigger:
```bash
curl -X POST http://localhost:8080/ \
  -H "Content-Type: application/json" \
  -d '{"commonEventObject": {"hostApp": "CALENDAR"}}'
```

---

## Part 4: Cloud Run Deployment with ko

Alexandria uses [ko](https://ko.build) for zero-Docker containerization on Cloud Run:

```bash
# 1. Set container registry
export KO_DOCKER_REPO=europe-west1-docker.pkg.dev/YOUR_PROJECT_ID/services

# 2. Build and deploy to Cloud Run
gcloud run deploy calendar-time-tracker \
  --image=$(ko build .) \
  --region=europe-west1 \
  --allow-unauthenticated \
  --set-env-vars="GEMINI_API_KEY=your-gemini-key"
```

Once deployed, copy the Cloud Run HTTPS service URL and register it as the HTTP endpoint for your Google Workspace Add-on deployment.

---

## Summary & Key Takeaways

1. **Start Simple:** Use Google Apps Script and Google AI Studio for instant, zero-cost prototyping without local setup.
2. **Standardized Interfaces:** Google Workspace Card JSON translates cleanly between Apps Script `CardService` and Go JSON schemas.
3. **Graduate Cleanly:** When moving to production, use Alexandria’s SRE Go stack (`go/platform/web`, `go/slog-gcp`, `go/retry`) and `ko` container builds.
