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

## Planned Work

| Order | Change | Tracking | Deliverable |
|-------|--------|----------|-------------|
| 10 | Decide workflow status transitions | [Issue #15](https://github.com/daikichiba9511/goat-cv/issues/15) | One state-machine specification shared by the product, design, and API documents |
| 11 | Implement workflow status transitions | Follow-up to [Issue #15](https://github.com/daikichiba9511/goat-cv/issues/15) | Image status APIs, review actions, UI, and behavior tests derived from the approved state machine |
| 12 | Add guideline management | [Issue #16](https://github.com/daikichiba9511/goat-cv/issues/16) | Guideline storage, API, safe Markdown rendering, and viewer panel |
| 13 | Add QA comments | [Issue #17](https://github.com/daikichiba9511/goat-cv/issues/17) | Image/Annotation comments with type and resolved state |
| 14 | Design pre-inference integration | [Issue #18](https://github.com/daikichiba9511/goat-cv/issues/18) | Provider-independent API contract and candidate lifecycle decision |
| 15 | Add pre-inference candidates | Follow-up to [Issue #18](https://github.com/daikichiba9511/goat-cv/issues/18) | Model API integration and accept, edit, and discard workflows derived from the approved design |
| 16 | Decide collaboration architecture | [Issue #19](https://github.com/daikichiba9511/goat-cv/issues/19) | Comparison and decision for the first collaboration and conflict boundary |
| 17 | Add initial realtime sync | Follow-up to [Issue #19](https://github.com/daikichiba9511/goat-cv/issues/19) | Narrow realtime synchronization implementation derived from the architecture decision |

## Dependency Order

- Completed [Issue #11](https://github.com/daikichiba9511/goat-cv/issues/11) provides the annotation validation boundary used by completed [Issue #13](https://github.com/daikichiba9511/goat-cv/issues/13).
- Completed [Issue #9](https://github.com/daikichiba9511/goat-cv/issues/9) provides object and label inspection before the remaining drawing tools are added.
- Completed [Issue #13](https://github.com/daikichiba9511/goat-cv/issues/13) provides the atomic save boundary used by completed [Issue #10](https://github.com/daikichiba9511/goat-cv/issues/10) and [Issue #12](https://github.com/daikichiba9511/goat-cv/issues/12).
- Issues [#15](https://github.com/daikichiba9511/goat-cv/issues/15), [#18](https://github.com/daikichiba9511/goat-cv/issues/18), and [#19](https://github.com/daikichiba9511/goat-cv/issues/19) are decision work. Each creates smaller implementation issues only after its behavior or architecture is explicit.

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

- [Issue #15: Decide workflow status transitions](https://github.com/daikichiba9511/goat-cv/issues/15), followed by implementation issues
- [Issue #16: Add guideline management](https://github.com/daikichiba9511/goat-cv/issues/16)
- [Issue #17: Add QA comments](https://github.com/daikichiba9511/goat-cv/issues/17)

Completion criteria:

- One state machine defines the status behavior across the product, design, API, implementation, and tests
- Image status can move only through transitions allowed by the documented workflow
- Guideline pages can be stored and viewed inside the annotator UI
- Comments can be attached to an Image or Annotation and marked resolved

### M4: Pre-Inference

Goal: let model output accelerate manual annotation without making predictions authoritative.

Work items:

- [Issue #18: Design pre-inference integration](https://github.com/daikichiba9511/goat-cv/issues/18)
- Implementation issues created from the approved contract and candidate lifecycle

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
- Keep PRs draft until the checks and manual smoke path are described

## Current Product Baseline

The `main` branch supports the single-user synchronous workflow: image upload, BBox and Polygon drawing, save/load, label assignment, Annotation Inspector, transform controls, GOAT JSON, COCO, and YOLO export, Edge APIs, and editing of `reading_order`, `key_value`, and `table_cell` relations.
