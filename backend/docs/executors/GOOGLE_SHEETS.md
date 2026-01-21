# Google Sheets Executor

## Overview

The Google Sheets executor provides integration with Google Sheets API v4, allowing workflows to read, write, and append data to Google Spreadsheets using service account credentials.

## Features

- **Read Operation**: Read data from specified ranges in Google Sheets
- **Write Operation**: Overwrite existing data in specified ranges
- **Append Operation**: Add new rows to the end of the sheet
- **Flexible Input Formats**: Supports 2D arrays, array of objects, JSON strings
- **Service Account Authentication**: Secure authentication using Google Cloud service account credentials
- **A1 Notation Support**: Standard spreadsheet range notation (e.g., A1:D10, Sheet1!A1:B5)

## Configuration

### Required Fields

| Field | Type | Description |
|-------|------|-------------|
| `operation` | string | Operation to perform: `read`, `write`, or `append` |
| `spreadsheet_id` | string | Google Sheets spreadsheet ID (from URL) |
| `credentials` | string | Service account credentials JSON |

### Optional Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `sheet_name` | string | "" | Name of the sheet tab (defaults to first sheet) |
| `range` | string | "" | Cell range in A1 notation (e.g., A1:D10) |
| `value_input_option` | string | "USER_ENTERED" | How to interpret input values: `RAW` or `USER_ENTERED` |
| `major_dimension` | string | "ROWS" | Data orientation: `ROWS` or `COLUMNS` |

## Operations

### Read Operation

Reads data from the spreadsheet and returns it as a 2D array.

**Configuration Example**:
```json
{
  "operation": "read",
  "spreadsheet_id": "1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgvE2upms",
  "sheet_name": "Sheet1",
  "range": "A1:D10",
  "credentials": "{\"type\":\"service_account\",\"project_id\":\"...\"}",
  "major_dimension": "ROWS"
}
```

**Output**:
```json
{
  "success": true,
  "operation": "read",
  "spreadsheet_id": "1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgvE2upms",
  "sheet_name": "Sheet1",
  "range": "A1:D10",
  "data": [
    ["Name", "Age", "City", "Email"],
    ["John Doe", "30", "New York", "john@example.com"],
    ["Jane Smith", "25", "London", "jane@example.com"]
  ],
  "row_count": 3,
  "column_count": 4,
  "metadata": {
    "major_dimension": "ROWS",
    "range": "Sheet1!A1:D10"
  },
  "duration_ms": 245
}
```

### Write Operation

Overwrites existing data in the specified range.

**Configuration Example**:
```json
{
  "operation": "write",
  "spreadsheet_id": "1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgvE2upms",
  "sheet_name": "Sheet1",
  "range": "A1:B2",
  "credentials": "{\"type\":\"service_account\",\"project_id\":\"...\"}",
  "value_input_option": "USER_ENTERED",
  "major_dimension": "ROWS"
}
```

**Input Data**:
```json
{
  "data": [
    ["Header1", "Header2"],
    ["Value1", "Value2"]
  ]
}
```

**Output**:
```json
{
  "success": true,
  "operation": "write",
  "spreadsheet_id": "1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgvE2upms",
  "sheet_name": "Sheet1",
  "range": "A1:B2",
  "updated_cells": 4,
  "updated_rows": 2,
  "updated_range": "Sheet1!A1:B2",
  "metadata": {
    "updated_columns": 2
  },
  "duration_ms": 312
}
```

### Append Operation

Adds new rows to the end of the sheet without overwriting existing data.

**Configuration Example**:
```json
{
  "operation": "append",
  "spreadsheet_id": "1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgvE2upms",
  "sheet_name": "Sheet1",
  "range": "A:D",
  "credentials": "{\"type\":\"service_account\",\"project_id\":\"...\"}",
  "value_input_option": "USER_ENTERED"
}
```

**Input Data**:
```json
{
  "data": [
    ["Alice Brown", "28", "Paris", "alice@example.com"],
    ["Bob Wilson", "35", "Berlin", "bob@example.com"]
  ]
}
```

**Output**:
```json
{
  "success": true,
  "operation": "append",
  "spreadsheet_id": "1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgvE2upms",
  "sheet_name": "Sheet1",
  "updated_cells": 8,
  "updated_rows": 2,
  "updated_range": "Sheet1!A11:D12",
  "metadata": {
    "updated_columns": 4
  },
  "duration_ms": 298
}
```

## Input Data Formats

The executor supports multiple input formats for write/append operations:

### 1. 2D Array (Direct)
```json
[
  ["row1col1", "row1col2"],
  ["row2col1", "row2col2"]
]
```

### 2. Array of Arrays
```json
{
  "data": [
    ["row1col1", "row1col2"],
    ["row2col1", "row2col2"]
  ]
}
```

### 3. Array of Objects
```json
[
  {"name": "John", "age": 30, "city": "NYC"},
  {"name": "Jane", "age": 25, "city": "LA"}
]
```
Note: Object keys order is not guaranteed. Use arrays for column order control.

### 4. JSON String
```json
{
  "json": "[[\"A\",\"B\"],[\"C\",\"D\"]]"
}
```

### 5. Map with Common Fields
The executor will try to extract data from these fields in order:
- `data`
- `values`
- `rows`
- `content`
- `body`

## Authentication

### Creating Service Account Credentials

1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Create or select a project
3. Enable Google Sheets API
4. Create a service account:
   - Go to "IAM & Admin" → "Service Accounts"
   - Click "Create Service Account"
   - Give it a name and description
   - Click "Create and Continue"
5. Create credentials:
   - Click on the created service account
   - Go to "Keys" tab
   - Click "Add Key" → "Create new key"
   - Choose JSON format
   - Download the JSON file
6. Share the spreadsheet with the service account email:
   - Copy the `client_email` from the JSON file
   - Open your Google Sheet
   - Click "Share"
   - Add the service account email with "Editor" permissions

### Credentials Format

```json
{
  "type": "service_account",
  "project_id": "your-project-id",
  "private_key_id": "key-id",
  "private_key": "-----BEGIN PRIVATE KEY-----\n...\n-----END PRIVATE KEY-----\n",
  "client_email": "your-service-account@your-project.iam.gserviceaccount.com",
  "client_id": "123456789",
  "auth_uri": "https://accounts.google.com/o/oauth2/auth",
  "token_uri": "https://oauth2.googleapis.com/token",
  "auth_provider_x509_cert_url": "https://www.googleapis.com/oauth2/v1/certs",
  "client_x509_cert_url": "https://www.googleapis.com/robot/v1/metadata/x509/..."
}
```

## Range Notation (A1 Notation)

### Basic Examples

- `A1` - Single cell
- `A1:D10` - Range from A1 to D10
- `A:A` - Entire column A
- `1:1` - Entire row 1
- `A:D` - Columns A through D
- `1:10` - Rows 1 through 10

### With Sheet Names

- `Sheet1!A1:D10` - Range in specific sheet
- `'My Sheet'!A1:B5` - Range in sheet with spaces (quotes required)

## Value Input Options

### USER_ENTERED (Default)

Values are parsed as if the user typed them into the Google Sheets UI:
- Numbers: `"123"` → number 123
- Dates: `"2024-01-15"` → date
- Formulas: `"=SUM(A1:A10)"` → formula
- Booleans: `"TRUE"` → boolean true

### RAW

Values are stored exactly as provided without parsing:
- Everything becomes strings
- Formulas are stored as text: `"=SUM(A1:A10)"` → text "=SUM(A1:A10)"

## Major Dimension

### ROWS (Default)

Data is organized by rows:
```json
[
  ["row1col1", "row1col2"],
  ["row2col1", "row2col2"]
]
```

### COLUMNS

Data is organized by columns:
```json
[
  ["col1row1", "col1row2"],
  ["col2row1", "col2row2"]
]
```

## Error Handling

The executor will return errors for:

- **Invalid credentials**: Malformed JSON or missing required fields
- **Authentication failure**: Invalid service account credentials
- **Permission denied**: Service account not shared with spreadsheet
- **Spreadsheet not found**: Invalid spreadsheet ID
- **Invalid range**: Malformed A1 notation
- **API quota exceeded**: Too many requests to Google Sheets API

## Performance Considerations

1. **Batch Operations**: For multiple writes, use write/append with multiple rows instead of multiple single-row operations
2. **Range Optimization**: Specify exact ranges instead of entire sheets when possible
3. **Credential Caching**: Service account credentials are cached per execution
4. **Rate Limits**: Google Sheets API has rate limits (100 requests per 100 seconds per user)

## Example Workflow

### Read → Transform → Write Pipeline

```mermaid
graph LR
    A[HTTP: Fetch Data] --> B[Google Sheets: Read]
    B --> C[Transform: Process]
    C --> D[Google Sheets: Append]
```

### Configuration Chain

1. **Read existing data**:
   - Operation: `read`
   - Range: `A1:D100`

2. **Transform data** (using Transform node):
   - Add calculations, filter rows, etc.

3. **Append results**:
   - Operation: `append`
   - Range: `E:H` (different columns)

## Best Practices

1. **Credentials Security**: Store credentials as environment variables or secrets, not in workflow config
2. **Sheet Sharing**: Always share spreadsheets with service account email
3. **Range Specificity**: Use specific ranges for better performance
4. **Error Recovery**: Implement retry logic for transient API failures
5. **Data Validation**: Validate data format before write/append operations
6. **Quota Management**: Monitor API usage to stay within quotas

## Limitations

1. **Cell Limits**: Maximum 10 million cells per spreadsheet
2. **Request Size**: Maximum 10 MB per request
3. **API Quotas**: 100 requests per 100 seconds per user
4. **Concurrent Writes**: Multiple simultaneous writes to same range may conflict

## Related Executors

- **HTTP Executor**: For REST API calls
- **Transform Executor**: For data transformation
- **CSV to JSON**: For converting CSV data before writing to sheets

## API Reference

Google Sheets API v4 Documentation:
- [REST API Reference](https://developers.google.com/sheets/api/reference/rest)
- [A1 Notation](https://developers.google.com/sheets/api/guides/concepts#cell)
- [Value Input Option](https://developers.google.com/sheets/api/reference/rest/v4/ValueInputOption)
