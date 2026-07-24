# API Design

## Conventions

- RESTful JSON API
- Base path: `/api/v1`
- レスポンス: 成功時は対象リソースを直接返す（リスト系は `{ "items": [...] }` ）
- エラー: `{ "error": "message" }`
- ID: UUID v7 (時系列ソート可能)
- タイムスタンプ: RFC 3339 (`2026-03-23T12:00:00Z`)

## Coordinate System

座標は **正規化座標 (0.0 - 1.0)** で保存・通信する。画像のピクセルサイズは Image エンティティが保持する。

| | 保存値 | 例 |
|---|---|---|
| BBox | `{ x, y, width, height }` (全て 0.0-1.0) | `{ x: 0.1, y: 0.2, width: 0.3, height: 0.05 }` |
| Polygon | `{ points: [{x, y}, ...] }` (全て 0.0-1.0) | `{ points: [{x: 0.1, y: 0.2}, ...] }` |
| Image | `width`, `height` (pixel) | `width: 2480, height: 3508` |

Frontend は `normalized * imagePixelSize` でキャンバス座標に変換する。
Zoom/Pan は Konva Stage の scale/position で制御し、座標変換には影響しない。

### Why normalized?

- 画像リサイズに対して座標が不変
- 異なる解像度の画像間で一貫性がある
- エクスポート時にピクセル座標への変換は `coord * width/height` で単純

## Affine Transform

Image 単位でアフィン変換（回転・反転）を適用し、表示を補正する。
座標は **変換後の座標空間** で記録する（annotator が見たまま = 座標）。

### Supported Transforms

| Transform | 値 | ユースケース |
|-----------|---|-------------|
| 回転 (90° 単位) | `rotation`: `0`, `90`, `180`, `270` | スキャン画像の向き補正 |
| 水平反転 | `flip_h`: `bool` | スキャンミス対応 |
| 垂直反転 | `flip_v`: `bool` | スキャンミス対応 |

任意角度の回転や skew は対象外（前処理で補正すべき）。

### Image Dimensions

Image は原画像と変換後の両方のサイズ情報を持つ。

| Field | 説明 |
|-------|------|
| `original_width`, `original_height` | 原画像のピクセルサイズ |
| `width`, `height` | 変換後のピクセルサイズ（90°/270°回転時に入れ替わる） |
| `rotation` | `0` / `90` / `180` / `270` |
| `flip_h`, `flip_v` | 水平/垂直反転 |

正規化座標は変換後の `width`, `height` に対する比率。

### Transform Pipeline

```
原画像ファイル → rotation適用 → flip適用 → 表示画像
                                            ↑
                                    この座標空間でアノテーション
```

- Backend: 画像ファイル自体は変換しない。変換メタデータのみ保存
- Frontend: Konva で rotation/flip を適用して表示し、その座標空間で操作
- 変換変更時: 既存アノテーションは無効になる（再アノテーションが必要）

## Image Serving

- アップロード: `POST /api/v1/projects/:projectId/images` (multipart/form-data)
- 配信: `GET /api/v1/images/:imageId/file` で画像バイナリを返す
- Frontend は `<img>` or Konva `Image` で直接読み込み
- サムネイル: Phase 1 では不要。必要になった時点で `?size=thumb` パラメータを追加

---

## Endpoints

### Projects

| Method | Path | Description | Phase |
|--------|------|-------------|-------|
| `POST` | `/projects` | プロジェクト作成 | 1 |
| `GET` | `/projects` | プロジェクト一覧 | 1 |
| `GET` | `/projects/:projectId` | プロジェクト詳細 | 1 |
| `PATCH` | `/projects/:projectId` | プロジェクト更新 | 1 |
| `DELETE` | `/projects/:projectId` | プロジェクト削除 | 1 |

```jsonc
// POST /projects
// Request
{ "name": "Invoice Annotation 2026Q1" }

// Response 201
{
  "id": "0194...",
  "name": "Invoice Annotation 2026Q1",
  "created_at": "2026-03-23T12:00:00Z"
}
```

### Label Definitions

| Method | Path | Description | Phase |
|--------|------|-------------|-------|
| `POST` | `/projects/:projectId/labels` | ラベル定義作成 | 1 |
| `GET` | `/projects/:projectId/labels` | ラベル定義一覧 | 1 |
| `PATCH` | `/projects/:projectId/labels/:labelId` | ラベル定義更新 | 1 |
| `DELETE` | `/projects/:projectId/labels/:labelId` | ラベル定義削除 | 1 |

```jsonc
// POST /projects/:projectId/labels
// Request
{
  "name": "header",
  "color": "#FF6B6B",
  "category": "object"
}

// Response 201
{
  "id": "0194...",
  "name": "header",
  "color": "#FF6B6B",
  "category": "object",
  "project_id": "0194..."
}
```

### Images

| Method | Path | Description | Phase |
|--------|------|-------------|-------|
| `POST` | `/projects/:projectId/images` | 画像アップロード (multipart) | 1 |
| `GET` | `/projects/:projectId/images` | 画像一覧 | 1 |
| `GET` | `/images/:imageId` | 画像メタ情報 | 1 |
| `GET` | `/images/:imageId/file` | 画像ファイル配信 | 1 |
| `PATCH` | `/images/:imageId` | 画像メタ更新 (status等) | 4 |
| `DELETE` | `/images/:imageId` | 画像削除 | 1 |

```jsonc
// POST /projects/:projectId/images
// Request: multipart/form-data, field "file"

// Response 201
{
  "id": "0194...",
  "filename": "invoice_001.png",
  "original_width": 2480,
  "original_height": 3508,
  "width": 2480,
  "height": 3508,
  "rotation": 0,
  "flip_h": false,
  "flip_v": false,
  "status": "pending",
  "project_id": "0194...",
  "uploaded_at": "2026-03-23T12:00:00Z"
}

// GET /projects/:projectId/images?status=pending
// Response 200
{
  "items": [
    {
      "id": "0194...",
      "filename": "invoice_001.png",
      "width": 2480,
      "height": 3508,
      "status": "pending",
      "project_id": "0194...",
      "uploaded_at": "2026-03-23T12:00:00Z"
    }
  ]
}
```

### Annotations

| Method | Path | Description | Phase |
|--------|------|-------------|-------|
| `POST` | `/images/:imageId/annotations` | アノテーション作成 | 1 |
| `GET` | `/images/:imageId/annotations` | アノテーション一覧 | 1 |
| `PATCH` | `/annotations/:annotationId` | アノテーション更新 | 1 |
| `DELETE` | `/annotations/:annotationId` | アノテーション削除 | 1 |
| `PUT` | `/images/:imageId/annotations` | 一括保存 (全置換) | 1 |

```jsonc
// POST /images/:imageId/annotations
// Request
{
  "type": "bbox",
  "coordinates": { "x": 0.1, "y": 0.2, "width": 0.3, "height": 0.05 },
  "label_id": "0194..."
}

// Response 201
{
  "id": "0194...",
  "type": "bbox",
  "coordinates": { "x": 0.1, "y": 0.2, "width": 0.3, "height": 0.05 },
  "label_id": "0194...",
  "image_id": "0194...",
  "created_at": "2026-03-23T12:00:00Z"
}

// PUT /images/:imageId/annotations (bulk save)
// Request
{
  "annotations": [
    {
      "id": "0194...",
      "type": "bbox",
      "coordinates": { "x": 0.1, "y": 0.2, "width": 0.3, "height": 0.05 },
      "label_id": "0194..."
    }
  ]
}

// Response 200
{
  "items": [...]
}
```

#### Coordinate Validation

| Annotation type または入力 | 受理条件 | API の振る舞い |
|----------------------------|----------|----------------|
| `bbox` | `x`、`y`、`width`、`height` が有限値であり、`width` と `height` が0より大きく、矩形全体が `0.0` から `1.0` に収まる | 受理する |
| `polygon` | `points` が相異なる点を3個以上持ち、各点の有限値 `x`、`y` が `0.0` から `1.0` に収まる | 受理する |
| 未対応type、Schema不一致、必須項目の欠落、未知の項目、非有限値、範囲外座標 | 該当なし | 検証理由を含む `400 Bad Request` を返す |
| 不正なAnnotationを含む `PUT` | 該当なし | 0始まりの `annotations[index]` を含む `400 Bad Request` を返し、既存Annotation集合を変更しない |

このPhaseではPolygonの自己交差を検証しない。

### Edges

| Method | Path | Description | Phase |
|--------|------|-------------|-------|
| `POST` | `/images/:imageId/edges` | エッジ作成 | 2 |
| `GET` | `/images/:imageId/edges` | エッジ一覧 | 2 |
| `DELETE` | `/edges/:edgeId` | エッジ削除 | 2 |
| `PUT` | `/images/:imageId/edges` | 一括保存 (全置換) | 2 |

```jsonc
// POST /images/:imageId/edges
// Request
{
  "source_annotation_id": "0194...",
  "target_annotation_id": "0194...",
  "type": "reading_order"
}

// Response 201
{
  "id": "0194...",
  "source_annotation_id": "0194...",
  "target_annotation_id": "0194...",
  "type": "reading_order",
  "image_id": "0194..."
}

// PUT /images/:imageId/edges (bulk save)
// Request
{
  "edges": [
    {
      "id": "0194...",
      "source_annotation_id": "0194...",
      "target_annotation_id": "0194...",
      "type": "reading_order"
    }
  ]
}

// Response 200
{
  "items": [...]
}
```

Validation:

- `source_annotation_id` と `target_annotation_id` は同一 `image_id` の Annotation を参照する
- self edge は拒否する
- 同一 `(source_annotation_id, target_annotation_id, type)` の重複は拒否する
- `reading_order` は閉路を作る Edge を拒否する
- `key_value` は `key` category から `value` category への 1:1 Edge のみ許可する
- `table_cell` は `table` category から `cell` category への 1:N Edge のみ許可する
- `PUT /images/:imageId/edges` は候補グラフ全体を検証し、不正な場合は既存 Edge を変更しない

### Image Annotation Graph

| Method | Path | Description | Phase |
|--------|------|-------------|-------|
| `PUT` | `/images/:imageId/graph` | AnnotationとEdgeの原子的な全置換 | 2 |

```jsonc
// PUT /images/:imageId/graph
// Request
{
  "annotations": [
    {
      "client_id": "temp-annotation-1",
      "id": "",
      "type": "bbox",
      "coordinates": { "x": 0.1, "y": 0.2, "width": 0.3, "height": 0.05 },
      "label_id": "0194..."
    },
    {
      "client_id": "existing-annotation-client-id",
      "id": "0195...",
      "type": "bbox",
      "coordinates": { "x": 0.6, "y": 0.2, "width": 0.3, "height": 0.05 },
      "label_id": "0194..."
    }
  ],
  "edges": [
    {
      "client_id": "temp-edge-1",
      "id": "",
      "source_annotation_client_id": "temp-annotation-1",
      "target_annotation_client_id": "existing-annotation-client-id",
      "type": "reading_order"
    }
  ]
}

// Response 200
{
  "annotations": [
    {
      "client_id": "temp-annotation-1",
      "annotation": {
        "id": "0194...",
        "image_id": "0194...",
        "type": "bbox",
        "coordinates": { "x": 0.1, "y": 0.2, "width": 0.3, "height": 0.05 },
        "label_id": "0194...",
        "created_at": "2026-03-23T12:00:00Z"
      }
    },
    {
      "client_id": "existing-annotation-client-id",
      "annotation": {
        "id": "0195...",
        "image_id": "0194...",
        "type": "bbox",
        "coordinates": { "x": 0.6, "y": 0.2, "width": 0.3, "height": 0.05 },
        "label_id": "0194...",
        "created_at": "2026-03-23T12:00:00Z"
      }
    }
  ],
  "edges": [
    {
      "client_id": "temp-edge-1",
      "edge": {
        "id": "0194...",
        "image_id": "0194...",
        "source_annotation_id": "0194...",
        "target_annotation_id": "0194...",
        "type": "reading_order"
      }
    }
  ]
}
```

Validation and transaction rules:

- `annotations` と `edges` は必須であり、空配列はImageのAnnotation Graphを空にする
- `client_id` は各配列内で一意なrequest-local IDとし、新規・既存Resourceの両方で必須とする
- `id` は既存の永続IDを更新する場合に送り、新規Resourceでは空文字列とする
- Edge端点は永続IDや配列位置ではなく、同じRequest内のAnnotation `client_id`を参照する
- Annotation、client参照、Edge集合をすべて検証してから、削除・Annotation挿入・Edge挿入を1つのDB Transactionで実行する
- 検証またはDB処理に失敗した場合、既存のAnnotationとEdgeをどちらも変更しない
- Responseの配列順は対応付けの契約に含めず、Clientは各Itemの`client_id`で永続Resourceを特定する

### Guidelines

| Method | Path | Description | Phase |
|--------|------|-------------|-------|
| `POST` | `/projects/:projectId/guidelines` | ガイドライン作成 | 4 |
| `GET` | `/projects/:projectId/guidelines` | ガイドライン一覧 | 4 |
| `GET` | `/guidelines/:guidelineId` | ガイドライン詳細 | 4 |
| `PATCH` | `/guidelines/:guidelineId` | ガイドライン更新 | 4 |
| `DELETE` | `/guidelines/:guidelineId` | ガイドライン削除 | 4 |

```jsonc
// POST /projects/:projectId/guidelines
// Request
{
  "title": "BBox Annotation Rules",
  "content": "## Rules\n\n- Include margins...",
  "order": 1
}

// Response 201
{
  "id": "0194...",
  "title": "BBox Annotation Rules",
  "content": "## Rules\n\n- Include margins...",
  "order": 1,
  "project_id": "0194...",
  "updated_at": "2026-03-23T12:00:00Z"
}
```

### Comments

| Method | Path | Description | Phase |
|--------|------|-------------|-------|
| `POST` | `/images/:imageId/comments` | コメント作成 | 4 |
| `GET` | `/images/:imageId/comments` | コメント一覧 | 4 |
| `PATCH` | `/comments/:commentId` | コメント更新 (resolve等) | 4 |
| `DELETE` | `/comments/:commentId` | コメント削除 | 4 |

```jsonc
// POST /images/:imageId/comments
// Request
{
  "body": "This bbox should include the header margin",
  "type": "qa_feedback",
  "annotation_id": "0194..."
}

// Response 201
{
  "id": "0194...",
  "author": "reviewer-1",
  "body": "This bbox should include the header margin",
  "type": "qa_feedback",
  "annotation_id": "0194...",
  "image_id": "0194...",
  "resolved": false,
  "created_at": "2026-03-23T12:00:00Z"
}
```

### Export

| Method | Path | Description | Phase |
|--------|------|-------------|-------|
| `GET` | `/projects/:projectId/export?format=json` | プロジェクト全体エクスポート | 1 |
| `GET` | `/images/:imageId/export?format=json` | 画像単位エクスポート | 1 |

#### Supported Formats

| Format | `format` param | 内容 | Phase |
|--------|---------------|------|-------|
| **GOAT JSON** | `json` | 独自JSON形式。全情報を完全に保持（エッジ含む） | 1 |
| **COCO** | `coco` | COCO Object Detection format | 3 |
| **YOLO** | `yolo` | YOLO txt format | 3 |

COCO/YOLO はノード（Annotation）のみ対応。エッジ（Reading Order, KV等）のグラフ構造は GOAT JSON でのみエクスポート可能。

#### GOAT JSON Format

```jsonc
// GET /projects/:projectId/export?format=json
{
  "format": "goat_json",
  "version": "1.0",
  "project": {
    "id": "0194...",
    "name": "Invoice Annotation 2026Q1"
  },
  "labels": [
    {
      "id": "0194...",
      "name": "header",
      "color": "#FF6B6B",
      "category": "object"
    }
  ],
  "images": [
    {
      "id": "0194...",
      "filename": "invoice_001.png",
      "original_width": 2480,
      "original_height": 3508,
      "width": 2480,
      "height": 3508,
      "rotation": 0,
      "flip_h": false,
      "flip_v": false,
      "annotations": [
        {
          "id": "0194...",
          "type": "bbox",
          "coordinates": { "x": 0.1, "y": 0.2, "width": 0.3, "height": 0.05 },
          "label_id": "0194...",
          "label": "header"
        }
      ],
      "edges": [
        {
          "id": "0194...",
          "source": "0194...",
          "target": "0194...",
          "type": "reading_order"
        }
      ]
    }
  ]
}
```

エクスポート JSON では `label_id` と `label` (名前) を両方含める。IDで正確な参照、名前で可読性を確保。

---

## Bulk Save Strategy

アノテーション作業は「画像を開いて複数のAnnotation/Edgeを編集し、保存する」というフローが基本。

2つの編集戦略を用意する:

| API | 用途 |
|-----|------|
| `POST/PATCH/DELETE` (個別) | リアルタイム自動保存、共同編集時の差分同期 |
| `PUT /images/:imageId/graph` | AnnotationとEdgeをまとめる明示的な保存操作 |

Annotator UIはGraph単位の`PUT`を使用し、AnnotationとEdgeのCollection別`PUT`を連続実行しない。
Collection別`PUT`はResource単位のAPIとして残るが、Annotation Graph全体の原子性は保証しない。
Phase 6 の共同編集で個別操作 + WebSocket 差分同期に発展させる。
