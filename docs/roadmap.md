# Roadmap

GOAT development proceeds through small branches and pull requests. Each phase can be split further when the diff becomes large.

The product scope is defined in [spec.md](spec.md). Architecture and API details are tracked in [design.md](design.md) and [api.md](api.md).

## Branch and PR Policy

- Base branch: `main` after the preceding roadmap PR is merged
- Stacked PRs may temporarily target the preceding branch when the base documentation PR is still open
- Branch naming: `codex/<short-description>`
- PRs start as draft PRs
- Each PR should include relevant checks in the description

## Planned Work

| Order | Branch | Scope | Deliverable |
|-------|--------|-------|-------------|
| 0 | `codex/prepare-public-repo` | Public repository preparation | Rename module path to `goat-cv`, add README, document roadmap, keep builds green |
| 1 | `codex/spec-and-roadmap` | Product planning | Product specification and PR-level roadmap |
| 2 | `codex/phase-2-edge-api` | Edge backend | Edge usecase validation, API handlers, tests |
| 3 | `codex/phase-2-edge-ui` | Edge UI | Reading Order edge drawing, edge display, save/load wiring |
| 4 | `codex/phase-2-polygon-api` | Polygon backend | Polygon coordinate validation and tests |
| 5 | `codex/phase-2-polygon-ui` | Polygon UI | Polygon drawing, editing, save/load |
| 6 | `codex/phase-3-export-formats` | Export | COCO and YOLO export for node annotations |
| 7 | `codex/phase-4-workflow-status` | Workflow | Image status transitions and review actions |
| 8 | `codex/phase-4-guidelines` | Guidelines | Guideline storage, API, and viewer panel |
| 9 | `codex/phase-4-comments` | QA comments | Image/Annotation comments and resolved state |
| 10 | `codex/phase-5-pre-inference` | Model assistance | Model API integration and prediction candidates |
| 11 | `codex/phase-6-collaboration-spike` | Collaboration | WebSocket/CRDT/OT architecture spike |
| 12 | `codex/phase-6-collaboration-sync` | Collaboration | Initial realtime synchronization implementation |

## Milestones

### M0: Public Development Setup

Goal: make the repository understandable and ready for PR-based development.

PRs:

- `codex/prepare-public-repo`
- `codex/spec-and-roadmap`

Completion criteria:

- Public repository exists
- README explains project identity and local development
- Specification defines scope and acceptance criteria
- Roadmap defines branch order and expected deliverables
- `go test ./...` and `npm run build` pass

### M1: Graph Annotation

Goal: support document graph datasets by connecting annotations with directed edges.

PRs:

- `codex/phase-2-edge-api`
- `codex/phase-2-edge-ui`
- `codex/phase-2-polygon-api`
- `codex/phase-2-polygon-ui`

Completion criteria:

- Edge APIs reject invalid cross-image, duplicate, self-referential, and cyclic reading-order edges
- Frontend can create, display, delete, save, and reload reading-order edges
- Frontend can create, edit, save, and reload Polygon annotations
- GOAT JSON includes BBox, Polygon, and Edge data

### M2: Export

Goal: provide common dataset export formats for downstream ML training.

PRs:

- `codex/phase-3-export-formats`

Completion criteria:

- GOAT JSON remains the complete export format
- COCO export supports BBox and Polygon annotations
- YOLO export supports BBox object detection annotations
- Export behavior is covered by backend tests

### M3: Workflow and QA

Goal: support review-oriented annotation workflows.

PRs:

- `codex/phase-4-workflow-status`
- `codex/phase-4-guidelines`
- `codex/phase-4-comments`

Completion criteria:

- Image status can move through the documented workflow
- Guideline pages can be stored and viewed inside the annotator UI
- Comments can be attached to an Image or Annotation and marked resolved

### M4: Pre-Inference

Goal: let model output accelerate manual annotation without making predictions authoritative.

PRs:

- `codex/phase-5-pre-inference`

Completion criteria:

- A configurable model API can return annotation candidates
- Candidates are visually distinct from saved annotations
- Users can accept, edit, or discard candidates

### M5: Collaboration

Goal: identify and implement the first safe collaborative editing model.

PRs:

- `codex/phase-6-collaboration-spike`
- `codex/phase-6-collaboration-sync`

Completion criteria:

- Collaboration approach is documented with tradeoffs
- The first realtime sync path is implemented for a narrow workflow
- Conflict behavior is explicit and testable

## Per-PR Checklist

- Update docs when behavior or public API changes
- Add or update tests for backend validation and export behavior
- Run `cd backend && go test ./...`
- Run `cd frontend && npm run build`
- Keep PRs draft until the checks and manual smoke path are described

## Done

Phase 1 implemented the single-user synchronous workflow: image upload, BBox drawing, save/load, label assignment, transform controls, and GOAT JSON export.
