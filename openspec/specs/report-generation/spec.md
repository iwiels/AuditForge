# Report Generation Specification

## Purpose

Define the format and structure of the final security audit report.

## Requirements

### Requirement: Markdown Output

The system MUST generate a consolidated report in Github-flavored Markdown format.

#### Scenario: Generate Report with Findings
- GIVEN a list of security findings with severity (High, Med, Low)
- WHEN the Reporter agent executes
- THEN it MUST create an `audit_report.md` file
- AND the report MUST contain a 'Summary' section and a 'Detailed Findings' section

### Requirement: Evidence Attachment

Each finding in the report SHOULD include clear evidence, such as the file path and line number of the vulnerability.

#### Scenario: Finding with Evidence
- GIVEN a SAST finding in `src/auth.py:L45`
- WHEN the report is generated
- THEN the finding entry MUST include the file link and a snippet or description of the location
