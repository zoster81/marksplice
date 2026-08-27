---
project: marksplice
owner: docs
---

# Project Guide

This guide tracks setup, release work, and troubleshooting for a small Go project.

## Setup

- [x] Install Go 1.26 or newer.
- [ ] Configure the documentation preview.

```go
doc, err := marksplice.Parse(source)
```

## Release plan

| Area | Status |
| --- | --- |
| API | Stable |
| Docs | In progress |

Review [Setup](#setup) first, then use [Troubleshooting](#troubleshooting) if an edit cannot be prepared.

## Troubleshooting

Keep the original source bytes when applying a prepared change.
