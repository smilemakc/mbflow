# Google Sheets Node - Quick Start Guide

## What is it?

A complete integration node for MBFlow that allows workflows to read, write, and append data to Google Sheets using service account authentication.

## Key Features

- **Read** data from Google Sheets
- **Write** (overwrite) data to Google Sheets
- **Append** new rows to Google Sheets
- Service account authentication (secure, server-side)
- Flexible input formats (arrays, objects, JSON strings)
- Full A1 notation support (Sheet1!A1:D10)

## Quick Setup

### 1. Get Service Account Credentials

1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Create/select project
3. Enable Google Sheets API
4. Create service account (IAM & Admin → Service Accounts)
5. Create JSON key
6. Share your spreadsheet with the service account email

### 2. Get Spreadsheet ID

From URL: `https://docs.google.com/spreadsheets/d/{SPREADSHEET_ID}/edit`

Copy the `{SPREADSHEET_ID}` part.

### 3. Configure Node in MBFlow

#### Read Data Example
```json
{
  "operation": "read",
  "spreadsheet_id": "YOUR_SPREADSHEET_ID",
  "sheet_name": "Sheet1",
  "range": "A1:D10",
  "credentials": "{paste service account JSON here}"
}
```

#### Write Data Example
```json
{
  "operation": "write",
  "spreadsheet_id": "YOUR_SPREADSHEET_ID",
  "sheet_name": "Sheet1",
  "range": "A1:B2",
  "credentials": "{paste service account JSON here}",
  "value_input_option": "USER_ENTERED"
}
```

Input data:
```json
{
  "data": [
    ["Name", "Age"],
    ["Alice", "30"]
  ]
}
```

#### Append Data Example
```json
{
  "operation": "append",
  "spreadsheet_id": "YOUR_SPREADSHEET_ID",
  "sheet_name": "Sheet1",
  "range": "A:B",
  "credentials": "{paste service account JSON here}"
}
```

Input data:
```json
[
  ["Bob", "25"],
  ["Charlie", "35"]
]
```

## Configuration Options

### Required
- **operation**: `read` | `write` | `append`
- **spreadsheet_id**: Your spreadsheet ID
- **credentials**: Service account JSON

### Optional
- **sheet_name**: Sheet tab name (default: first sheet)
- **range**: A1 notation range (e.g., "A1:D10", "A:D")
- **value_input_option**: `USER_ENTERED` (parse formulas) | `RAW` (as-is)
- **major_dimension**: `ROWS` | `COLUMNS`

## Output

### Read Operation
```json
{
  "success": true,
  "operation": "read",
  "data": [
    ["Header1", "Header2"],
    ["Value1", "Value2"]
  ],
  "row_count": 2,
  "column_count": 2,
  "duration_ms": 245
}
```

### Write/Append Operations
```json
{
  "success": true,
  "operation": "write",
  "updated_cells": 4,
  "updated_rows": 2,
  "updated_range": "Sheet1!A1:B2",
  "duration_ms": 312
}
```

## Common Use Cases

### 1. Report Generation
```
HTTP Node → Transform → Google Sheets (write)
```
Fetch data from API, transform it, write report to Google Sheets.

### 2. Data Collection
```
Telegram → Transform → Google Sheets (append)
```
Collect data from Telegram, format it, append to log sheet.

### 3. Data Processing
```
Google Sheets (read) → Transform → Google Sheets (write)
```
Read data, process it, write results to different range/sheet.

### 4. Daily Digest
```
Multiple Sources → Merge → Google Sheets (append)
```
Collect data from multiple sources, merge, append daily digest.

## Tips & Best Practices

1. **Always share spreadsheet** with service account email from credentials
2. **Use specific ranges** for better performance (not entire sheet)
3. **Store credentials securely** (environment variables, not in code)
4. **Monitor API quotas** (100 requests per 100 seconds)
5. **Use USER_ENTERED** for formulas and date parsing
6. **Use RAW** if you want exact string values

## Troubleshooting

### Error: "Permission denied"
→ Share spreadsheet with service account email

### Error: "Spreadsheet not found"
→ Check spreadsheet ID is correct

### Error: "Invalid credentials"
→ Verify JSON is valid service account credentials

### Error: "Invalid range"
→ Check A1 notation syntax (A1:D10, Sheet1!A1:B5)

## A1 Notation Examples

- `A1` - single cell
- `A1:D10` - range
- `A:A` - entire column A
- `1:1` - entire row 1
- `A:D` - columns A to D
- `Sheet1!A1:D10` - range in specific sheet

## Files Location

### Backend
- Executor: `/backend/pkg/executor/builtin/google_sheets.go`
- Tests: `/backend/pkg/executor/builtin/google_sheets_test.go`
- Docs: `/backend/docs/executors/GOOGLE_SHEETS.md`

### Frontend
- Component: `/mbflow-react/components/nodes/config/GoogleSheetsNodeConfig.tsx`
- Types: `/mbflow-react/types/nodeConfigs.ts`

## Documentation

- **Full Guide**: `/backend/docs/executors/GOOGLE_SHEETS.md`
- **Implementation Summary**: `/backend/docs/GOOGLE_SHEETS_IMPLEMENTATION.md`
- **This Quick Start**: `/GOOGLE_SHEETS_QUICKSTART.md`

## Support

- Google Sheets API: https://developers.google.com/sheets/api
- Service Accounts: https://developers.google.com/identity/protocols/oauth2/service-account
- A1 Notation: https://developers.google.com/sheets/api/guides/concepts#cell

---

**Ready to use!** Just add the node from NodeLibrary → Actions → Google Sheets
