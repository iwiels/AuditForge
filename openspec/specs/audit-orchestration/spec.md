# Audit Orchestration Specification

## Purpose

Define the central coordination of the security audit lifecycle using a multi-agent state machine.

## Requirements

### Requirement: Graph Sequence

The system MUST execute audit phases in the following order: Scout (Recon) -> Audit Dispatch -> Evidence Collection -> Analysis -> Reporting.

#### Scenario: Successful Audit Flow
- GIVEN a target project path
- WHEN the orchestrator starts
- THEN the Scout agent identifies the tech stack
- AND the Dispatcher triggers the relevant auditors
- AND the final state contains a consolidated evidence list

### Requirement: State Persistence

The system SHALL maintain the audit state through the entire execution to allow for resumability and correlation between nodes.

#### Scenario: Audit Resumption
- GIVEN an audit that was interrupted during the Audit Dispatch phase
- WHEN the orchestrator is restarted with the same target
- THEN it SHOULD resume from the last successful state (Scout data preserved)
