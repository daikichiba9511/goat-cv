# Roadmap

GOAT development proceeds through small branches and pull requests. Each phase can be split further when the diff becomes large.

The product scope is defined in [spec.md](spec.md). Architecture and API details are tracked in [design.md](design.md) and [api.md](api.md).

## Branch and PR Policy

- Base branch: `main` after the preceding roadmap PR is merged
- Stacked PRs may temporarily target the preceding branch when the base documentation PR is still open
- Branch names should describe the change, not the author or tool, for example `docs/spec-and-roadmap` or `backend/edge-api`
- PRs start as draft PRs
- Each PR should be one self-contained change
- Each PR description should explain what changed, why it changed, important context, and validation

## Planned Work

| Order | Change | Scope | Deliverable |
|-------|--------|-------|-------------|
| 0 | Prepare public repository | Public repository preparation | Rename module path to `goat-cv`, add README, document roadmap, keep builds green |
| 1 | Add product specification and roadmap | Product planning | Product specification and PR-level roadmap |
| 2 | Add edge API validation | Edge backend | Edge usecase validation, API handlers, tests |
| 3 | Add edge annotation UI | Edge UI | Reading Order edge drawing, edge display, save/load wiring |
| 4 | Add polygon API validation | Polygon backend | Polygon coordinate validation and tests |
| 5 | Add polygon annotation UI | Polygon UI | Polygon drawing, editing, save/load |
| 6 | Add COCO and YOLO export | Export | COCO and YOLO export for node annotations |
| 7 | Add workflow status transitions | Workflow | Image status transitions and review actions |
| 8 | Add guideline management | Guidelines | Guideline storage, API, and viewer panel |
| 9 | Add QA comments | QA comments | Image/Annotation comments and resolved state |
| 10 | Add pre-inference candidates | Model assistance | Model API integration and prediction candidates |
| 11 | Decide collaboration architecture | Collaboration | WebSocket/CRDT/OT architecture spike |
| 12 | Add initial realtime sync | Collaboration | Initial realtime synchronization implementation |

## Milestones

### M0: Public Development Setup

Goal: make the repository understandable and ready for PR-based development.

PRs:

- Prepare public repository
- Add product specification and roadmap

Completion criteria:

- Public repository exists
- README explains project identity and local development
- Specification defines scope and acceptance criteria
- Roadmap defines branch order and expected deliverables
- `go test ./...` and `npm run build` pass

### M1: Graph Annotation

Goal: support graph-structured CV datasets by connecting annotations with directed edges.

PRs:

- Add edge API validation
- Add edge annotation UI
- Add polygon API validation
- Add polygon annotation UI

Completion criteria:

- Edge APIs reject invalid cross-image, duplicate, self-referential, and cyclic reading-order edges
- Frontend can create, display, delete, save, and reload reading-order edges
- Frontend can create, edit, save, and reload Polygon annotations
- GOAT JSON includes BBox, Polygon, and Edge data

### M2: Export

Goal: provide common dataset export formats for downstream ML training.

PRs:

- Add COCO and YOLO export

Completion criteria:

- GOAT JSON remains the complete export format
- COCO export supports BBox and Polygon annotations
- YOLO export supports BBox object detection annotations
- Export behavior is covered by backend tests

### M3: Workflow and QA

Goal: support review-oriented annotation workflows.

PRs:

- Add workflow status transitions
- Add guideline management
- Add QA comments

Completion criteria:

- Image status can move through the documented workflow
- Guideline pages can be stored and viewed inside the annotator UI
- Comments can be attached to an Image or Annotation and marked resolved

### M4: Pre-Inference

Goal: let model output accelerate manual annotation without making predictions authoritative.

PRs:

- Add pre-inference candidates

Completion criteria:

- A configurable model API can return annotation candidates
- Candidates are visually distinct from saved annotations
- Users can accept, edit, or discard candidates

### M5: Collaboration

Goal: identify and implement the first safe collaborative editing model.

PRs:

- Decide collaboration architecture
- Add initial realtime sync

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
