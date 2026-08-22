/**
 * Time Tracker AI — Google Calendar Add-on (Level 1: Apps Script)
 *
 * Scaffolds an AI-powered time writing sidebar directly inside Google Calendar.
 * Uses Google AI Studio (Gemini 1.5/2.0 Flash) with zero external infrastructure.
 */

const SCRIPT_PROP_KEY = 'GEMINI_API_KEY';

/**
 * Homepage trigger for Google Calendar sidebar.
 */
function onCalendarHomepage(e) {
  const userProps = PropertiesService.getUserProperties();
  const apiKey = userProps.getProperty(SCRIPT_PROP_KEY);

  const card = CardService.newCardBuilder();
  card.setHeader(
    CardService.newCardHeader()
      .setTitle('AI Time Tracker')
      .setSubtitle('Log billable project time with Gemini AI')
      .setImageUrl('https://www.gstatic.com/images/icons/material/system/1x/schedule_black_24dp.png')
  );

  const section = CardService.newCardSection();

  if (!apiKey) {
    section.addWidget(
      CardService.newTextParagraph().setText(
        '🔑 <b>Step 1: Set your Gemini API Key</b><br>Get your free key with 0 friction at <a href="https://aistudio.google.com">aistudio.google.com</a>.'
      )
    );

    const apiKeyInput = CardService.newTextInput()
      .setFieldName('api_key')
      .setTitle('Gemini API Key')
      .setHint('Paste key from Google AI Studio');
    section.addWidget(apiKeyInput);

    const saveKeyAction = CardService.newAction().setFunctionName('onSaveApiKey');
    section.addWidget(
      CardService.newTextButton()
        .setText('Save API Key')
        .setOnClickAction(saveKeyAction)
        .setTextButtonStyle(CardService.TextButtonStyle.FILLED)
    );

    card.addSection(section);
    return card.build();
  }

  // Main Time Logging Interface
  section.addWidget(
    CardService.newTextParagraph().setText(
      '📝 <b>Describe your work:</b><br>Type a quick note in plain English. Gemini will extract the client, project, duration, and summary.'
    )
  );

  const noteInput = CardService.newTextInput()
    .setFieldName('work_note')
    .setTitle('Work Note')
    .setMultiline(true)
    .setHint('e.g. Spent 1.5 hours debugging OAuth token refresh for Acme Corp');
  section.addWidget(noteInput);

  const analyzeAction = CardService.newAction().setFunctionName('onAnalyzeNote');
  section.addWidget(
    CardService.newTextButton()
      .setText('✨ Analyze & Structure with AI')
      .setOnClickAction(analyzeAction)
      .setTextButtonStyle(CardService.TextButtonStyle.FILLED)
  );

  card.addSection(section);

  // Settings section
  const settingsSection = CardService.newCardSection().setCollapsible(true).setHeader('Settings');
  const clearKeyAction = CardService.newAction().setFunctionName('onClearApiKey');
  settingsSection.addWidget(
    CardService.newTextButton().setText('Reset API Key').setOnClickAction(clearKeyAction)
  );
  card.addSection(settingsSection);

  return card.build();
}

/**
 * Saves the Gemini API key in UserProperties.
 */
function onSaveApiKey(e) {
  const apiKey = e.formInputs && e.formInputs.api_key ? e.formInputs.api_key[0].trim() : '';
  if (!apiKey) {
    return CardService.newActionResponseBuilder()
      .setNotification(CardService.newNotification().setText('Please enter a valid API key.'))
      .build();
  }

  PropertiesService.getUserProperties().setProperty(SCRIPT_PROP_KEY, apiKey);

  const nav = CardService.newNavigation().updateCard(onCalendarHomepage(e));
  return CardService.newActionResponseBuilder()
    .setNavigation(nav)
    .setNotification(CardService.newNotification().setText('✅ API key saved successfully!'))
    .build();
}

/**
 * Clears the saved Gemini API key.
 */
function onClearApiKey(e) {
  PropertiesService.getUserProperties().deleteProperty(SCRIPT_PROP_KEY);
  const nav = CardService.newNavigation().updateCard(onCalendarHomepage(e));
  return CardService.newActionResponseBuilder().setNavigation(nav).build();
}

/**
 * Calls Gemini API to parse natural language work note and structure the time entry.
 */
function onAnalyzeNote(e) {
  const note = e.formInputs && e.formInputs.work_note ? e.formInputs.work_note[0].trim() : '';
  if (!note) {
    return CardService.newActionResponseBuilder()
      .setNotification(CardService.newNotification().setText('Please enter a work note first.'))
      .build();
  }

  const apiKey = PropertiesService.getUserProperties().getProperty(SCRIPT_PROP_KEY);
  if (!apiKey) {
    return onCalendarHomepage(e);
  }

  try {
    const timeEntry = callGeminiExtract(note, apiKey);
    return CardService.newActionResponseBuilder()
      .setNavigation(CardService.newNavigation().pushCard(buildConfirmCard(timeEntry)))
      .build();
  } catch (err) {
    return CardService.newActionResponseBuilder()
      .setNotification(CardService.newNotification().setText('AI analysis failed: ' + err.message))
      .build();
  }
}

/**
 * Factory function creating an immutable TimeEntry object (Crockford style).
 */
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

/**
 * Calls Google AI Studio Gemini API with JSON instruction.
 */
function callGeminiExtract(note, apiKey) {
  const url = 'https://generativelanguage.googleapis.com/v1beta/models/gemini-1.5-flash:generateContent?key=' + encodeURIComponent(apiKey);

  const prompt = `You are a professional time-tracking assistant.
Analyze the following work note and extract:
- client: (string, the client name or "Internal" if not specified)
- project: (string, the project name or feature)
- duration_hours: (number, duration in hours, e.g. 0.5, 1.0, 1.5, default to 1.0 if not stated)
- title: (string, short title e.g. "[Client] Task Name")
- summary: (string, professional 1-2 sentence description suitable for a timesheet/invoice)

Work note: "${note}"

Respond with ONLY raw JSON matching this schema, without markdown formatting:
{
  "client": "string",
  "project": "string",
  "duration_hours": 1.0,
  "title": "string",
  "summary": "string"
}`;

  const payload = {
    contents: [
      {
        parts: [{ text: prompt }]
      }
    ],
    generationConfig: {
      temperature: 0.2,
      responseMimeType: "application/json"
    }
  };

  const response = UrlFetchApp.fetch(url, {
    method: 'post',
    contentType: 'application/json',
    payload: JSON.stringify(payload),
    muteHttpExceptions: true
  });

  const statusCode = response.getResponseCode();
  if (statusCode !== 200) {
    throw new Error('Gemini API error (HTTP ' + statusCode + '): ' + response.getContentText());
  }

  const resJson = JSON.parse(response.getContentText());
  const rawText = resJson.candidates[0].content.parts[0].text;
  const parsed = JSON.parse(rawText);
  return makeTimeEntry(parsed.client, parsed.project, parsed.duration_hours, parsed.title, parsed.summary);
}

/**
 * Builds the confirmation and edit card before logging to Google Calendar.
 */
function buildConfirmCard(timeEntry) {
  const card = CardService.newCardBuilder();
  card.setHeader(
    CardService.newCardHeader()
      .setTitle('Confirm Time Entry')
      .setSubtitle(timeEntry.title || 'Review extracted details')
  );

  const section = CardService.newCardSection();

  section.addWidget(
    CardService.newTextInput()
      .setFieldName('entry_title')
      .setTitle('Event Title')
      .setValue(timeEntry.title || `[${timeEntry.client}] ${timeEntry.project}`)
  );

  section.addWidget(
    CardService.newTextInput()
      .setFieldName('entry_duration')
      .setTitle('Duration (Hours)')
      .setValue(String(timeEntry.duration_hours || 1.0))
  );

  section.addWidget(
    CardService.newTextInput()
      .setFieldName('entry_summary')
      .setTitle('Description / Notes')
      .setMultiline(true)
      .setValue(timeEntry.summary || '')
  );

  const logAction = CardService.newAction().setFunctionName('onLogToCalendar');
  section.addWidget(
    CardService.newTextButton()
      .setText('📅 Create Calendar Event')
      .setOnClickAction(logAction)
      .setTextButtonStyle(CardService.TextButtonStyle.FILLED)
  );

  card.addSection(section);
  return card.build();
}

const CALENDAR_NAME = 'Time Tracker';

/**
 * Gets or creates the dedicated 'Time Tracker' Google Calendar.
 */
function getOrCreateTimeTrackerCalendar() {
  const cals = CalendarApp.getCalendarsByName(CALENDAR_NAME);
  if (cals.length > 0) {
    return cals[0];
  }
  return CalendarApp.createCalendar(CALENDAR_NAME, {
    summary: 'Dedicated calendar for AI Time Tracker entries'
  });
}

/**
 * Creates the event directly in the dedicated Google Calendar.
 */
function onLogToCalendar(e) {
  const formInputs = e.formInputs || {};
  const title = formInputs.entry_title ? formInputs.entry_title[0] : 'Time Log';
  const durationHours = parseFloat(formInputs.entry_duration ? formInputs.entry_duration[0] : '1.0') || 1.0;
  const summary = formInputs.entry_summary ? formInputs.entry_summary[0] : '';

  const now = new Date();
  const startTime = new Date(now.getTime() - (durationHours * 60 * 60 * 1000));
  const endTime = now;

  try {
    const cal = getOrCreateTimeTrackerCalendar();
    const event = cal.createEvent(title, startTime, endTime, {
      description: summary + '\n\n— Logged with AI Time Tracker'
    });

    // Tag with a distinctive calendar color (e.g. Cyan = 7 or Peacock = 6)
    event.setColor(CalendarApp.EventColor.CYAN);

    const successCard = CardService.newCardBuilder()
      .setHeader(CardService.newCardHeader().setTitle('✅ Time Logged!'))
      .addSection(
        CardService.newCardSection()
          .addWidget(
            CardService.newTextParagraph().setText(
              `<b>${title}</b><br>Duration: <b>${durationHours}h</b><br>Logged to dedicated <b>${CALENDAR_NAME}</b> calendar.`
            )
          )
          .addWidget(
            CardService.newTextButton()
              .setText('Log Another Entry')
              .setOnClickAction(
                CardService.newAction().setFunctionName('onResetToHomepage')
              )
          )
      )
      .build();

    return CardService.newActionResponseBuilder()
      .setNavigation(CardService.newNavigation().popToRoot().updateCard(successCard))
      .setNotification(CardService.newNotification().setText(`Event created in ${CALENDAR_NAME} calendar!`))
      .build();
  } catch (err) {
    return CardService.newActionResponseBuilder()
      .setNotification(CardService.newNotification().setText('Failed to create event: ' + err.message))
      .build();
  }
}

/**
 * Resets navigation back to the fresh homepage card.
 */
function onResetToHomepage(e) {
  return CardService.newActionResponseBuilder()
    .setNavigation(CardService.newNavigation().popToRoot().updateCard(onCalendarHomepage(e)))
    .build();
}
