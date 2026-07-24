# Roadmap

GOAT development proceeds through small branches and pull requests. Each phase can be split further when the diff becomes large.

The product scope is defined in [spec.md](spec.md). Architecture and API details are tracked in [design.md](design.md) and [api.md](api.md).

## Branch and PR Policy

- Base branch: the latest `main`
- Start each branch after its dependency PR is merged; do not stack work on an unmerged branch
- Branch names should describe the change, not the author or tool, for example `docs/spec-and-roadmap` or `backend/edge-api`
- Open a normal PR after local validation is complete
- Each PR should be one self-contained change
- Each PR description should explain what changed, why it changed, important context, and validation

## Completed Work

| Order | Change | Evidence | Result |
|-------|--------|----------|--------|
| 0 | Prepare public repository | [PR #1](https://github.com/daikichiba9511/goat-cv/pull/1) | Published the `goat-cv` repository with the renamed module, README, and local build instructions |
| 1 | Add product specification and roadmap | [PR #2](https://github.com/daikichiba9511/goat-cv/pull/2), [PR #3](https://github.com/daikichiba9511/goat-cv/pull/3) | Defined the product scope, clarified the general CV focus, and established the PR-level roadmap |
| 2 | Add edge API validation | [PR #5](https://github.com/daikichiba9511/goat-cv/pull/5), [PR #8](https://github.com/daikichiba9511/goat-cv/pull/8) | Added Edge APIs and validation for cross-image, duplicate, self-referential, and cyclic reading-order edges |
| 3 | Add edge annotation UI | [PR #6](https://github.com/daikichiba9511/goat-cv/pull/6), [PR #7](https://github.com/daikichiba9511/goat-cv/pull/7), [PR #8](https://github.com/daikichiba9511/goat-cv/pull/8) | Added label-aware BBox presentation and Reading Order edge drawing, display, save, reload, and deletion |
| 4 | Validate annotation coordinates | [PR #22](https://github.com/daikichiba9511/goat-cv/pull/22) | Added atomic API validation for normalized Bounding Box and Polygon coordinate schemas |
| 5 | Save the image graph atomically | [PR #24](https://github.com/daikichiba9511/goat-cv/pull/24) | Added one transactional save contract for Annotations and Edges with explicit client ID mapping |
| 6 | Add annotation inspector | [PR #26](https://github.com/daikichiba9511/goat-cv/pull/26) | Added synchronized Annotation listing, filtering, relationship counts, selection, and deletion outside the Canvas |
| 7 | Add remaining edge relation UI | [PR #27](https://github.com/daikichiba9511/goat-cv/pull/27) | Added constraint-aware creation, display, selection, deletion, save, and reload for all three Edge relation types |
| 8 | Add polygon annotation UI | [PR #28](https://github.com/daikichiba9511/goat-cv/pull/28) | Added Polygon drawing, vertex editing, Edge connections, deletion, save, and reload |
| 9 | Add COCO and YOLO export | [PR #29](https://github.com/daikichiba9511/goat-cv/pull/29) | Added self-contained COCO and YOLO archives with transform-aware coordinates, reproducible class mappings, and explicit exclusions |
| 10 | Decide workflow status transitions | [PR #32](https://github.com/daikichiba9511/goat-cv/pull/32) | Defined lifecycle and escalation as orthogonal states with explicit events, guards, allowed operations, and error responses |
| 11 | Persist workflow state transitions | [PR #37](https://github.com/daikichiba9511/goat-cv/pull/37) | Added lifecycle and escalation persistence, event-driven Usecase transitions, explicit conflicts, and fail-fast schema migration |
| 11 | Add workflow API and mutation guards | [PR #38](https://github.com/daikichiba9511/goat-cv/pull/38) | Added the event endpoint, combined status filters, state-aware Graph and transform guards, and explicit conflict responses |
| 11 | Add Annotator workflow controls | [PR #39](https://github.com/daikichiba9511/goat-cv/pull/39) | Added visible workflow state and actions, combined Image filters, save-before-transition ordering, and state-aware editing controls |
| 11 | Verify the workflow HTTP contract | [PR #40](https://github.com/daikichiba9511/goat-cv/pull/40) | Added production-router integration scenarios for approval, revision, escalation, errors, guards, Comments, and combined filters |
| 12 | Add guideline management | [PR #30](https://github.com/daikichiba9511/goat-cv/pull/30) | Added Project-scoped Guideline CRUD and safe Markdown viewing without losing Canvas editing state |
| 13 | Add QA comments | [PR #31](https://github.com/daikichiba9511/goat-cv/pull/31) | Added Image/Annotation QA Comments with selected-object filtering, resolved state, and retained audit history |

## Planned Work

| Order | Change | Tracking | Deliverable |
|-------|--------|----------|-------------|
| 14 | Design pre-inference integration | [Issue #18](https://github.com/daikichiba9511/goat-cv/issues/18) | Provider-independent API contract and candidate lifecycle decision |
| 15 | Persist pre-inference state | [Issue #41](https://github.com/daikichiba9511/goat-cv/issues/41) | Inference Run, Candidate, and Label Mapping persistence |
| 15 | Add provider-neutral run API | [Issue #42](https://github.com/daikichiba9511/goat-cv/issues/42) | Transformed-image inference, idempotency, failure classification, and public API |
| 15 | Save candidate decisions atomically | [Issue #43](https://github.com/daikichiba9511/goat-cv/issues/43) | Candidate decisions committed with the Image Graph |
| 15 | Add Annotator candidate workflow | [Issue #44](https://github.com/daikichiba9511/goat-cv/issues/44) | Distinct candidate display and staged accept, edit, and discard actions |
| 15 | Add generic HTTP provider | [Issue #45](https://github.com/daikichiba9511/goat-cv/issues/45) | Configured multipart adapter with bounded responses and secret isolation |
| 16 | Decide collaboration architecture | [Issue #19](https://github.com/daikichiba9511/goat-cv/issues/19) | Comparison and decision for the first collaboration and conflict boundary |
| 17 | Add initial realtime sync | Follow-up to [Issue #19](https://github.com/daikichiba9511/goat-cv/issues/19) | Narrow realtime synchronization implementation derived from the architecture decision |

## Dependency Order

- Completed [Issue #11](https://github.com/daikichiba9511/goat-cv/issues/11) provides the annotation validation boundary used by completed [Issue #13](https://github.com/daikichiba9511/goat-cv/issues/13).
- Completed [Issue #9](https://github.com/daikichiba9511/goat-cv/issues/9) provides object and label inspection before the remaining drawing tools are added.
- Completed [Issue #13](https://github.com/daikichiba9511/goat-cv/issues/13) provides the atomic save boundary used by completed [Issue #10](https://github.com/daikichiba9511/goat-cv/issues/10) and [Issue #12](https://github.com/daikichiba9511/goat-cv/issues/12).
- Completed [Issue #15](https://github.com/daikichiba9511/goat-cv/issues/15) defines the behavior, completed [Issue #33](https://github.com/daikichiba9511/goat-cv/issues/33) persists it, and completed [Issue #34](https://github.com/daikichiba9511/goat-cv/issues/34) provides the API and mutation guards used by completed [Issue #35](https://github.com/daikichiba9511/goat-cv/issues/35) and completed [Issue #36](https://github.com/daikichiba9511/goat-cv/issues/36).
- [Issue #18](https://github.com/daikichiba9511/goat-cv/issues/18) defines the Pre-Inference contract used by [Issue #41](https://github.com/daikichiba9511/goat-cv/issues/41), [Issue #42](https://github.com/daikichiba9511/goat-cv/issues/42), [Issue #43](https://github.com/daikichiba9511/goat-cv/issues/43), [Issue #44](https://github.com/daikichiba9511/goat-cv/issues/44), and [Issue #45](https://github.com/daikichiba9511/goat-cv/issues/45).
- [Issue #41](https://github.com/daikichiba9511/goat-cv/issues/41) is the persistence base for [Issue #42](https://github.com/daikichiba9511/goat-cv/issues/42) and [Issue #43](https://github.com/daikichiba9511/goat-cv/issues/43). [Issue #44](https://github.com/daikichiba9511/goat-cv/issues/44) starts after both are merged, while [Issue #45](https://github.com/daikichiba9511/goat-cv/issues/45) starts after [Issue #42](https://github.com/daikichiba9511/goat-cv/issues/42).
- [Issue #19](https://github.com/daikichiba9511/goat-cv/issues/19) remains decision work and creates smaller implementation issues only after its architecture is explicit.

## Milestones

### M0: Public Development Setup

Goal: make the repository understandable and ready for PR-based development.

Completed PRs:

- [Prepare public repository](https://github.com/daikichiba9511/goat-cv/pull/1)
- [Add product specification and roadmap](https://github.com/daikichiba9511/goat-cv/pull/2)
- [Clarify the general CV product scope](https://github.com/daikichiba9511/goat-cv/pull/3)

Completion criteria:

- Public repository exists
- README explains project identity and local development
- Specification defines scope and acceptance criteria
- Roadmap defines branch order and expected deliverables
- `go test ./...` and `npm run build` pass

### M1: Graph Annotation

Goal: support graph-structured CV datasets by connecting annotations with directed edges.

Work items:

- Completed: Edge API validation in [PR #5](https://github.com/daikichiba9511/goat-cv/pull/5)
- Completed: Reading Order edge UI in [PR #7](https://github.com/daikichiba9511/goat-cv/pull/7)
- Completed: Annotation coordinate validation in [PR #22](https://github.com/daikichiba9511/goat-cv/pull/22)
- Completed: Atomic image graph save in [PR #24](https://github.com/daikichiba9511/goat-cv/pull/24)
- Completed: Annotation Inspector in [PR #26](https://github.com/daikichiba9511/goat-cv/pull/26)
- Completed: Remaining Edge relation UI in [PR #27](https://github.com/daikichiba9511/goat-cv/pull/27)
- Completed: Polygon annotation UI in [PR #28](https://github.com/daikichiba9511/goat-cv/pull/28)

Completion criteria:

- Edge APIs reject invalid cross-image, duplicate, self-referential, and cyclic reading-order edges
- Annotation APIs reject malformed or out-of-range Bounding Box and Polygon coordinates without partial replacement
- Annotation and Edge changes are saved as one image-level transaction
- Frontend can create, display, delete, save, and reload `reading_order`, `key_value`, and `table_cell` edges
- Frontend can inspect the Label, shape type, and relationships of each annotation outside the Canvas
- Frontend can create, edit, save, and reload Polygon annotations
- GOAT JSON includes BBox, Polygon, and Edge data

### M2: Export

Goal: provide common dataset export formats for downstream ML training.

Work item:

- Completed: COCO and YOLO export in [PR #29](https://github.com/daikichiba9511/goat-cv/pull/29)

Completion criteria:

- GOAT JSON remains the complete export format
- COCO export supports BBox and Polygon annotations
- YOLO export supports BBox object detection annotations
- Export behavior is covered by backend tests

### M3: Workflow and QA

Goal: support review-oriented annotation workflows.

Work items:

- Completed: Workflow status transition decision in [PR #32](https://github.com/daikichiba9511/goat-cv/pull/32)
- Completed: Persistence and Usecase transitions in [PR #37](https://github.com/daikichiba9511/goat-cv/pull/37)
- Completed: Workflow API and mutation guards in [PR #38](https://github.com/daikichiba9511/goat-cv/pull/38)
- Completed: Annotator workflow controls in [PR #39](https://github.com/daikichiba9511/goat-cv/pull/39)
- Completed: Focused workflow contract tests in [PR #40](https://github.com/daikichiba9511/goat-cv/pull/40)
- Completed: Guideline management in [PR #30](https://github.com/daikichiba9511/goat-cv/pull/30)
- Completed: QA Comment management in [PR #31](https://github.com/daikichiba9511/goat-cv/pull/31)

Completion criteria:

- One state machine defines the status behavior across the product, design, API, implementation, and tests
- Image status can move only through transitions allowed by the documented workflow
- Annotator displays current workflow state and allowed actions while disabling state-prohibited edits
- Guideline pages can be stored and viewed inside the annotator UI
- Comments can be attached to an Image or Annotation and marked resolved

### M4: Pre-Inference

Goal: let model output accelerate manual annotation without making predictions authoritative.

Work items:

- [Issue #18: Design pre-inference integration](https://github.com/daikichiba9511/goat-cv/issues/18)
- [Issue #41: Persist Inference Run, Candidate, and Label Mapping](https://github.com/daikichiba9511/goat-cv/issues/41)
- [Issue #42: Add provider-neutral Inference Run API](https://github.com/daikichiba9511/goat-cv/issues/42)
- [Issue #43: Save Candidate decisions with the Image Graph](https://github.com/daikichiba9511/goat-cv/issues/43)
- [Issue #44: Add the Annotator Candidate workflow](https://github.com/daikichiba9511/goat-cv/issues/44)
- [Issue #45: Add the Generic HTTP Provider Adapter](https://github.com/daikichiba9511/goat-cv/issues/45)

Completion criteria:

- A configurable model API can return annotation candidates
- Candidates are visually distinct from saved annotations
- Users can accept, edit, or discard candidates

### M5: Collaboration

Goal: identify and implement the first safe collaborative editing model.

Work items:

- [Issue #19: Decide collaboration architecture](https://github.com/daikichiba9511/goat-cv/issues/19)
- An initial realtime synchronization issue created from the approved architecture

Completion criteria:

- Collaboration approach is documented with tradeoffs
- The first realtime sync path is implemented for a narrow workflow
- Conflict behavior is explicit and testable

## Per-PR Checklist

- Update docs when behavior or public API changes
- Link the tracking issue and close it only when its acceptance criteria are met
- Add or update tests for backend validation and export behavior
- Run `cd backend && go test ./...`
- Run `cd frontend && npm test`
- Run `cd frontend && npm run build`
- Open the PR only after checks and the manual smoke path are described

## Current Product Baseline

The `main` branch supports the single-user synchronous workflow: image upload, BBox and Polygon drawing, save/load, label assignment, Annotation Inspector, lifecycle and escalation actions with filtered Image lists and state-aware editing controls, Project Guideline and QA Comment management with safe Markdown viewing, transform controls, GOAT JSON, COCO, and YOLO export, Edge APIs, and editing of `reading_order`, `key_value`, and `table_cell` relations.
