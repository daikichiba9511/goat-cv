# GOAT: Go CV Annotation Tool - Design Document

## Overview

画像データセット作成のためのComputer Visionアノテーションアプリケーション。
帳票・ドキュメント画像は初期の重点ユースケースとして扱う。
非同期共同編集・事前推論を最終目標とし、まずは同期的・単一ユーザーで小さく始める。

### Target Domain

- **対象画像**: 一般的な画像。初期重点ユースケースは帳票、フォーム、請求書等のドキュメント画像
- **アノテーションタスク**:
  1. **Object Detection** — 画像内のオブジェクトをBBox/Polygonで検出
  2. **Reading Order** — オブジェクト間に有向エッジを貼り、読み順を定義（有向グラフ構造）
  3. **Table Analysis** — テーブル領域の検出とセル構造の定義（親子関係エッジ）
  4. **Information Extraction** — エンティティ領域にセマンティックラベル付与（例: 日付、金額、会社名）
  5. **KV Extraction** — Key領域とValue領域を検出し、対応関係をエッジで定義

## Tech Stack

| Category | Selection | ADR |
|----------|-----------|-----|
| Backend HTTP | Chi | [ADR-0001](adr/0001-go-http-framework.md) |
| Database / Query | SQLite + sqlc | [ADR-0002](adr/0002-database-and-query.md) |
| Canvas | Konva (react-konva) | [ADR-0003](adr/0003-canvas-library.md) |
| State Management | Zustand | [ADR-0004](adr/0004-state-management.md) |
| Styling | Tailwind CSS | [ADR-0005](adr/0005-styling.md) |
| Frontend Framework | React + Vite | - |
| Language | Go (backend) / TypeScript (frontend) | - |

## Architecture

```
┌─────────────────────────────────────────────────┐
│                   Frontend                       │
│  React + Vite                                    │
│  ┌───────────┐  ┌──────────┐  ┌──────────────┐  │
│  │  Toolbar   │  │ Sidebar  │  │ Annotation   │  │
│  │ (tools)    │  │ (images, │  │ Canvas       │  │
│  │            │  │  labels) │  │ (Konva)      │  │
│  └───────────┘  └──────────┘  └──────────────┘  │
│         │              │              │          │
│         └──────────────┼──────────────┘          │
│                        │                         │
│                  Zustand Store                    │
└────────────────────────┼─────────────────────────┘
                         │ REST API (JSON)
┌────────────────────────┼─────────────────────────┐
│                   Backend                        │
│  Go + Chi                                        │
│  ┌──────────┐  ┌──────────┐  ┌──────────────┐   │
│  │ Handler  │→ │ Usecase  │→ │ Repository   │   │
│  │ (HTTP)   │  │ (Logic)  │  │ (SQLite/sqlc)│   │
│  └──────────┘  └──────────┘  └──────────────┘   │
│                                      │           │
│                              ┌───────┴────────┐  │
│                              │ SQLite   Local  │  │
│                              │ (.db)    FS     │  │
│                              └────────────────┘  │
└──────────────────────────────────────────────────┘
```

### Backend Layers

| Layer | Responsibility |
|-------|---------------|
| **Handler** | HTTPリクエスト/レスポンス変換、バリデーション |
| **Usecase** | ビジネスロジック、複数Repositoryの協調 |
| **Repository** | データ永続化、SQLクエリ実行 |
| **Domain** | ドメインモデル定義（他レイヤーに依存しない） |

依存方向: Handler → Usecase → Repository(interface) ← Repository(impl)

## Domain Model

```
Project 1 ---* LabelDefinition
   |                |
   ├─ id            ├─ id
   ├─ name          ├─ name (e.g. "header", "invoice_number", "table")
   └─ created_at    ├─ color
                    └─ category: "object" | "entity" | "key" | "value" | "table" | "cell"

Project 1 ---* Guideline
   |                |
   |                ├─ id
   |                ├─ title
   |                ├─ content (Markdown)
   |                ├─ order (表示順)
   |                └─ updated_at

Project 1 ---* Image 1 ---* Annotation
                  |         |    |
                  ├─ id     |    ├─ id
                  ├─ file   |    ├─ type: "bbox" | "polygon"
                  ├─ original_width  |  ├─ coordinates: JSON (normalized, post-transform space)
                  ├─ original_height |  ├─ label_id → LabelDefinition
                  ├─ width  |    └─ created_at
                  ├─ height |
                  ├─ rotation: 0|90|180|270
                  ├─ flip_h |
                  ├─ flip_v |
                  ├─ status |
                  └─ ...    |
                            ├──* Edge
                            |     |
                            |     ├─ id
                            |     ├─ source_annotation_id
                            |     ├─ target_annotation_id
                            |     └─ type: "reading_order" | "key_value" | "table_cell"
                            |
                            └──* Comment (QA / Escalation)
                                  |
                                  ├─ id
                                  ├─ author
                                  ├─ body (Markdown)
                                  ├─ type: "qa_feedback" | "escalation"
                                  ├─ annotation_id (nullable, 特定Annotationへの指摘)
                                  ├─ resolved: bool
                                  └─ created_at
```

### Image Status (Workflow)

```
 ┌──────────┐    annotator     ┌───────────┐    reviewer     ┌──────────┐
 │  pending  │ ──────────────→ │ annotated │ ──────────────→ │ in_review│
 └──────────┘    completes     └───────────┘    picks up     └──────────┘
                                     ↑                            │
                                     │ rejected (差戻し)           │
                                     └────────────────────────────┤
                                                                  │ approved
                                                             ┌────▼─────┐
                                                             │ approved │
                                                             └──────────┘

 Any status ──→ escalated (判断に迷った場合、上位者に相談)
               ↑              │
               │              │ resolved (回答後、元のstatusに戻る)
               └──────────────┘
```

### Concepts

- **Annotation** — 画像上の領域（ノード）。BBox/Polygonで位置を定義し、LabelDefinitionでセマンティクスを付与
- **Edge** — Annotation間の有向関係（辺）。typeによって関係の意味が変わる
- **LabelDefinition** — プロジェクト単位で定義するラベル体系。categoryでタスク種別を区別
- **Guideline** — プロジェクト単位のアノテーションマニュアル。Markdown形式、複数ページ構成
- **Comment** — Image/Annotationに対するQAフィードバックやエスカレーション

Image単位でAnnotation(ノード) + Edge(辺) の有向グラフを構成する。

### Coordinate System

座標は **正規化座標 (0.0 - 1.0)** で保存・通信する。変換後の Image `width/height` (px) に対する比率。

- Frontend: `normalized * pixelSize` でキャンバス座標に変換
- エクスポート: `coord * width/height` でピクセル座標に変換
- Zoom/Pan は Konva Stage 側で制御し、座標値に影響しない

### Affine Transform

Image 単位で回転・反転を適用し、スキャン画像の表示を補正する。

- **適用順序**: `rotation → flip_h → flip_v`（api.md と統一）
- **座標空間**: アノテーションは変換後の座標空間で記録（annotator が見たまま = 座標）
- **width/height**: 変換後のピクセルサイズ（90°/270° 回転時に original と入れ替わる）
- **変換変更時**: 既存アノテーションは無効になるため再アノテーションが必要。変換変更は作業開始前に確定させる運用を推奨

詳細は [api.md](api.md#affine-transform) を参照。

### Annotation Types

- **BBox**: `{ x, y, width, height }` — 全て有限な 0.0-1.0 の正規化座標。`width > 0`、`height > 0` かつ矩形全体が正規化画像空間に収まる
- **Polygon**: `{ points: [{x, y}, ...] }` — 全て有限な 0.0-1.0 の正規化座標。相異なる点を3個以上持つ

Annotation type と座標 Schema が一致しない入力は Usecase で永続化前に拒否する。
一括保存では全件を検証してから Repository を呼び出し、1件でも不正な場合は既存 Annotation を変更しない。
Polygon の自己交差判定は初期の座標検証に含めない。

### Export Formats

| Format | 内容 | Phase |
|--------|------|-------|
| **GOAT JSON** | 独自JSON形式。全情報を完全に保持（アノテーション + エッジ + 変換情報） | 1 |
| **COCO** | COCO Object Detection format（ノードのみ） | 3 |
| **YOLO** | YOLO txt format（ノードのみ） | 3 |

エッジ（Reading Order, KV等）のグラフ構造は GOAT JSON でのみエクスポート可能。
詳細は [api.md](api.md#export) を参照。

### Edge Types

| Type | Meaning | Example |
|------|---------|---------|
| `reading_order` | 読み順 (source → targetの順) | テキストブロック間の読み順 |
| `key_value` | KVペア (source=Key, target=Value) | "氏名" → "山田太郎" |
| `table_cell` | テーブル親子 (source=Table, target=Cell) | テーブル領域 → 各セル |

### Edge Constraints

- **同一 Image 内のみ**: source と target は同じ image_id に属する Annotation でなければならない
- **Edge type ごとの制約**:

| Type | source の label category | target の label category | 多重度 | グラフ構造 |
|------|------------------------|------------------------|--------|-----------|
| `reading_order` | any | any | source → N targets, N sources → target | DAG（有向非巡回） |
| `key_value` | `key` | `value` | 1:1 | 独立ペア |
| `table_cell` | `table` | `cell` | 1 table → N cells | 木（1階層） |

- **自己参照禁止**: source_annotation_id ≠ target_annotation_id
- **重複禁止**: 同一 (source, target, type) の組み合わせは1つのみ
- **巡回禁止**: reading_order は DAG であること（バリデーションで閉路検出）

#### reading_order: DAG を許容する理由

段組み（2カラムレイアウト等）で1ノードから複数ノードへ分岐するケースが帳票では実際にある。
合流（複数 source → 1 target）も同様に許容する。巡回のみ禁止。

#### key_value: 1:1 の運用ルール

- 1つの Key に対して Value は1つの BBox/Polygon で囲む
- 複数の値が1領域に収まっている場合（例: "担当者: 山田, 田中"）は Value BBox を1つにする
- 物理的に離れた複数値は、同じ label の別 KV ペアとして扱う

### Task-to-Model Mapping

| Task | Annotationの使い方 | Edgeの使い方 | Label category |
|------|-------------------|-------------|----------------|
| Object Detection | BBox/Polygonで検出 | - | `object` |
| Reading Order | 検出済みオブジェクト間 | `reading_order` | - |
| Table Analysis | テーブル/セルをBBox | `table_cell` | `table`, `cell` |
| Information Extraction | エンティティ領域にラベル | - | `entity` |
| KV Extraction | Key/Value領域をBBox | `key_value` | `key`, `value` |

## UI Layout & Navigation

```
┌─────────────────────────────────────────────────────────────────────┐
│  Header: Project名 / Image (3/120) / Status: annotated             │
├──────────┬──────────────────────────────────────┬───────────────────┤
│          │                                      │                   │
│ Sidebar  │         Canvas (Konva)               │  Right Panel      │
│          │                                      │                   │
│ ・Image  │   ┌─────────────────────────┐        │ [Tab: Labels]     │
│   List   │   │                         │        │  ・label list     │
│          │   │   Image                 │        │  ・assign label   │
│ ・Filter │   │   + Annotations         │        │                   │
│  by      │   │   + Edges               │        │ [Tab: Guideline]  │
│  status  │   │                         │        │  ・manual viewer  │
│          │   └─────────────────────────┘        │  (Markdown)       │
│          │                                      │                   │
│          │  Toolbar: BBox / Polygon / Edge /    │ [Tab: Comments]   │
│          │           Select / Pan               │  ・QA feedback    │
│          │                                      │  ・Escalation     │
│          │                                      │  ・thread形式     │
└──────────┴──────────────────────────────────────┴───────────────────┘
```

### Navigation Flows

- **アノテーション作業中にマニュアル参照**: Right PanelのGuidelineタブで即座に確認。Canvas作業を中断しない
- **QAフィードバック確認**: Right PanelのCommentsタブ。特定Annotationへの指摘はクリックでCanvasにハイライト
- **エスカレーション起票**: Commentsタブから起票。画像/特定Annotationにピン留め可能
- **ステータスフィルタ**: Sidebarで`pending` / `rejected`等でフィルタし、作業対象を素早く特定
- **レビュー画面**: 同じAnnotator画面を使い、reviewer権限でapprove/rejectボタンを表示

## Directory Structure

```
goat-cv/
├── backend/
│   ├── cmd/
│   │   └── server/
│   │       └── main.go
│   ├── internal/
│   │   ├── handler/
│   │   │   ├── project.go
│   │   │   ├── image.go
│   │   │   └── annotation.go
│   │   ├── domain/
│   │   │   ├── project.go
│   │   │   ├── image.go
│   │   │   ├── annotation.go
│   │   │   ├── edge.go
│   │   │   ├── label.go
│   │   │   ├── guideline.go
│   │   │   └── comment.go
│   │   ├── repository/
│   │   │   └── sqlite/
│   │   │       ├── project.go
│   │   │       ├── image.go
│   │   │       ├── annotation.go
│   │   │       ├── edge.go
│   │   │       ├── label.go
│   │   │       ├── guideline.go
│   │   │       └── comment.go
│   │   └── usecase/
│   │       ├── project.go
│   │       ├── annotation.go
│   │       └── label.go
│   ├── db/
│   │   ├── migrations/
│   │   └── queries/
│   ├── storage/                  # image files (.gitignore)
│   ├── go.mod
│   └── go.sum
│
├── frontend/
│   ├── src/
│   │   ├── api/
│   │   ├── components/
│   │   │   ├── canvas/
│   │   │   │   ├── AnnotationCanvas.tsx
│   │   │   │   ├── BBoxTool.tsx
│   │   │   │   ├── PolygonTool.tsx
│   │   │   │   └── EdgeTool.tsx
│   │   │   ├── sidebar/
│   │   │   ├── toolbar/
│   │   │   └── right-panel/
│   │   │       ├── LabelPanel.tsx
│   │   │       ├── GuidelinePanel.tsx
│   │   │       └── CommentPanel.tsx
│   │   ├── stores/
│   │   │   ├── annotationStore.ts
│   │   │   └── projectStore.ts
│   │   ├── types/
│   │   ├── pages/
│   │   │   ├── ProjectList.tsx
│   │   │   └── Annotator.tsx
│   │   ├── App.tsx
│   │   └── main.tsx
│   ├── package.json
│   ├── tsconfig.json
│   ├── vite.config.ts
│   └── tailwind.config.ts
│
├── docs/
│   ├── design.md
│   └── adr/
├── tasks/
└── README.md
```

## Phased Roadmap

| Phase | Scope | Key Deliverables |
|-------|-------|-----------------|
| **1** | Single user, sync | 画像アップロード、BBox描画・保存・読込 |
| **2** | Graph annotation | Reading Order エッジ描画・保存、Polygon対応 |
| **3** | Label & export | ラベル管理、エクスポート（COCO等 + グラフ構造） |
| **4** | Workflow & QA | ガイドライン、Image status管理、QAコメント、エスカレーション |
| **5** | Pre-inference | モデルAPI連携、推論結果をアノテーション候補として表示 |
| **6** | Collaborative editing | WebSocket + CRDT/OTによる非同期共同編集 |
