# Google Calendar Time Tracker Add-on (Level 1: Apps Script)

Zero-friction, zero-infrastructure Google Calendar Add-on for logging project time using free Google AI Studio (Gemini 1.5/2.0 Flash) and Google Apps Script.

## What It Does

1. Adds an **AI Time Tracker** sidebar to Google Calendar.
2. Accepts natural language work notes (e.g., *"Spent 1.5 hours debugging database migrations for Acme"*).
3. Calls Gemini AI to structure the note into a typed time entry (`client`, `project`, `duration_hours`, `summary`).
4. Creates a color-coded time block event directly in a dedicated **Time Tracker** Google Calendar (automatically created or linked).

## Quick Setup (In-Browser in 3 Minutes)

1. **(Optional) Create a Dedicated Calendar in Google Calendar**:
   - In [calendar.google.com](https://calendar.google.com), click **+** next to *Other calendars* > **Create new calendar**.
   - Name it `Time Tracker` (or let the script auto-create it for you!).

1. **Get Free Gemini API Key**:
   - Go to [aistudio.google.com](https://aistudio.google.com) and sign in with any Google account.
   - Click **Get API key** and copy your personal key.

2. **Create Apps Script Project**:
   - Go to [script.google.com](https://script.google.com) and click **New Project**.
   - Rename the project to `Calendar Time Tracker AI`.

3. **Configure Manifest (`appsscript.json`)**:
   - In the Apps Script editor, click **Project Settings** (gear icon) > check **Show "appsscript.json" manifest file in editor**.
   - Open `appsscript.json` and replace its entire contents with [`appsscript.json`](appsscript.json).

4. **Add Code (`Code.gs`)**:
   - Open `Code.gs` and replace its contents with [`Code.gs`](Code.gs).
   - Click **Save** (disk icon).

5. **Test in Google Calendar**:
   - Click **Deploy > Test deployments**.
   - Click **Install test deployment**.
   - Open [calendar.google.com](https://calendar.google.com).
   - In the right-side companion bar, click the **Time Tracker AI** icon (schedule icon).
   - Paste your Gemini API key, write a work note, and click **Analyze & Structure with AI**!

## Architecture

```
[ Google Calendar Companion Bar ]
               │
               ▼
   [ CardService UI (Code.gs) ]
               │
      ┌────────┴────────┐
      ▼                 ▼
[ Gemini Flash API ]  [ CalendarApp ]
 (aistudio free key)   (Creates Event)
```
