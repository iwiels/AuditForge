# Asset Discovery Specification

## Purpose

Describe how the system identifies the target project's technology stack, structure, and potential attack surface.

## Requirements

### Requirement: Stack Detection

The Scout agent MUST identify the primary programming languages and package managers present in the target directory.

#### Scenario: Identify Node.js Project
- GIVEN a directory containing a `package.json` file
- WHEN the Scout agent runs
- THEN it MUST report the stack as 'javascript/typescript'
- AND identify `npm` or `yarn` as the package manager

#### Scenario: Identify Python Project
- GIVEN a directory containing `pyproject.toml` or `requirements.txt`
- WHEN the Scout agent runs
- THEN it MUST report the stack as 'python'
