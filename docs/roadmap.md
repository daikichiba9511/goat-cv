# Roadmap

GOAT development proceeds through small branches and pull requests. Each phase can be split further when the diff becomes large.

## Branch and PR Policy

- Base branch: `main`
- Branch naming: `codex/<short-description>`
- PRs start as draft PRs
- Each PR should include relevant checks in the description

## Planned Work

| Order | Branch | Scope | Deliverable |
|-------|--------|-------|-------------|
| 0 | `codex/prepare-public-repo` | Public repository preparation | Rename module path to `goat-cv`, add README, document roadmap, keep builds green |
| 1 | `codex/phase-2-edge-annotation` | Graph annotation | Reading Order edge drawing, edge persistence, edge display |
| 2 | `codex/phase-2-polygon-tool` | Polygon annotation | Polygon drawing, editing, save/load |
| 3 | `codex/phase-3-export-formats` | Export | COCO and YOLO export for node annotations |
| 4 | `codex/phase-4-workflow-status` | Workflow | Image status transitions and review actions |
| 5 | `codex/phase-4-guidelines-comments` | QA support | Guideline panel, QA comments, escalation comments |
| 6 | `codex/phase-5-pre-inference` | Model assistance | Model API integration and prediction candidates |
| 7 | `codex/phase-6-collaboration` | Collaboration | WebSocket sync and collaboration architecture spike |

## Done

Phase 1 implemented the single-user synchronous workflow: image upload, BBox drawing, save/load, label assignment, transform controls, and GOAT JSON export.
