// Package orchestrator provides the security audit orchestrator.
// This file re-exports marker types from the markers package for convenience.
package orchestrator

import "orquestador-auditor/internal/markers"

// ContentMarker is re-exported from the markers package.
type ContentMarker = markers.ContentMarker

// InjectWithMarkers is re-exported from the markers package.
var InjectWithMarkers = markers.InjectWithMarkers

// FormatMarkdownSection is re-exported from the markers package.
var FormatMarkdownSection = markers.FormatMarkdownSection

// ListMarkedComponents is re-exported from the markers package.
var ListMarkedComponents = markers.ListMarkedComponents
