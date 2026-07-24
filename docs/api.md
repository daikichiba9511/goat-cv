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

Image 単位でアフィン変換（回転と反転）を適用し、表示を補正する。
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
原画像ファイル → source軸のflip_h/flip_v → 時計回りrotation → 表示画像
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
| `PATCH` | `/images/:imageId` | transformメタデータ更新 | 1 |
| `POST` | `/images/:imageId/workflow-transitions` | workflow eventの適用 | 4 |
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
  "escalated": false,
  "project_id": "0194...",
  "uploaded_at": "2026-03-23T12:00:00Z"
}

// GET /projects/:projectId/images?status=pending&escalated=false
// Response 200
{
  "items": [
    {
      "id": "0194...",
      "filename": "invoice_001.png",
      "width": 2480,
      "height": 3508,
      "status": "pending",
      "escalated": false,
      "project_id": "0194...",
      "uploaded_at": "2026-03-23T12:00:00Z"
    }
  ]
}
```

`status`と`escalated`を同時に指定したImage一覧は、両方の条件を満たすImageだけを返す。
`PATCH /images/:imageId`は`rotation`、`flip_h`、`flip_v`だけを受け付け、workflow状態を直接上書きしない。

#### Workflow Transitions

Workflow状態は任意の値で更新せず、[Image Workflow Status Specification](workflow-status.md#状態機械)に定義したeventを適用する。

```jsonc
// POST /images/:imageId/workflow-transitions
// Request
{
  "event": "review_started"
}

// Response 200
{
  "id": "0194...",
  "status": "in_review",
  "escalated": false
}
```

未知のeventは`400 Bad Request`、対象Imageが存在しない場合は`404 Not Found`を返す。
既知のeventでも現在状態から許可されない場合は、Imageを変更せず`409 Conflict`を返す。

```jsonc
// Response 409
{
  "error": "workflow transition not allowed",
  "current": {
    "status": "approved",
    "escalated": false
  },
  "allowed_events": ["approval_reopened"]
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
| `PUT` | `/images/:imageId/graph` | Annotation、Edge、PreLabel判断の原子的な保存 | 2, 5 |

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
  ],
  "pre_label_decisions": [
    {
      "pre_label_id": "0196...",
      "decision": "accept",
      "annotation_client_id": "temp-annotation-1"
    },
    {
      "pre_label_id": "0197...",
      "decision": "discard"
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
  ],
  "pre_label_decisions": [
    {
      "pre_label_id": "0196...",
      "state": "accepted",
      "accepted_annotation_id": "0194...",
      "decided_at": "2026-07-24T12:05:00Z"
    },
    {
      "pre_label_id": "0197...",
      "state": "discarded",
      "accepted_annotation_id": null,
      "decided_at": "2026-07-24T12:05:00Z"
    }
  ]
}
```

Validation and transaction rules:

- `annotations` と `edges` は必須であり、空配列はImageのAnnotation Graphを空にする
- `client_id` は各配列内で一意なrequest-local IDとし、新規・既存Resourceの両方で必須とする
- `id` は既存の永続IDを更新する場合に送り、新規Resourceでは空文字列とする
- Edge端点は永続IDや配列位置ではなく、同じRequest内のAnnotation `client_id`を参照する
- Annotation、client参照、Edge集合はRepositoryを呼ぶ前に検証する
- `pre_label_decisions`は省略または空配列を許可し、その場合はPreLabelを変更しない
- `accept`は同じRequest内の新規Annotation `client_id`を1つ参照し、`discard`はAnnotationを参照しない
- PreLabelは同じImageのCurrent Pre-Labelsに属する`pending`状態でなければならない
- PreLabelのImport、transform、stateに関するguardはTransaction内でも再確認し、Annotation、Edge、PreLabel判断を1つのDB Transactionで実行する
- 検証またはDB処理に失敗した場合、既存のAnnotation、Edge、PreLabelを変更しない
- Responseの配列順は対応付けの契約に含めず、Clientは各Itemの`client_id`で永続Resourceを特定する

### Pre-Label Import

Pre-Label Importの責務境界、Import Schema、再取り込み、判断状態は[Pre-Label Import Specification](pre-label-import.md)に従う。

| Method | Path | Description | Phase |
|--------|------|-------------|-------|
| `POST` | `/projects/:projectId/pre-label-imports` | Project単位のPreLabelImport作成 | 5 |
| `GET` | `/images/:imageId/pre-labels` | ImageのCurrent Pre-Labels取得 | 5 |

#### Create Import

```jsonc
// POST /projects/:projectId/pre-label-imports
// Request
{
  "format": "goat_pre_labels",
  "version": "1.0",
  "source": {
    "name": "layout-detector",
    "version": "2026-07-24.1",
    "reference": "batch-2026-07-24"
  },
  "images": [
    {
      "image_id": "0194...",
      "coordinate_space": {
        "width": 2480,
        "height": 3508,
        "rotation": 0,
        "flip_h": false,
        "flip_v": false
      },
      "items": [
        {
          "external_id": "detection-42",
          "type": "bbox",
          "coordinates": {
            "x": 0.10,
            "y": 0.20,
            "width": 0.30,
            "height": 0.05
          },
          "confidence": 0.94,
          "source_label": {
            "key": "0",
            "name": "header"
          },
          "label_id": "0195..."
        }
      ]
    }
  ]
}

// Response 201
{
  "import": {
    "id": "0198...",
    "project_id": "0193...",
    "source": {
      "name": "layout-detector",
      "version": "2026-07-24.1",
      "reference": "batch-2026-07-24"
    },
    "imported_at": "2026-07-24T12:00:00Z"
  },
  "images": [
    {
      "image_id": "0194...",
      "pre_labels": [
        {
          "id": "0199...",
          "import_id": "0198...",
          "image_id": "0194...",
          "external_id": "detection-42",
          "type": "bbox",
          "coordinates": {
            "x": 0.10,
            "y": 0.20,
            "width": 0.30,
            "height": 0.05
          },
          "confidence": 0.94,
          "source_label": {
            "key": "0",
            "name": "header"
          },
          "label_id": "0195...",
          "state": "pending",
          "accepted_annotation_id": null,
          "decided_at": null
        }
      ]
    }
  ]
}
```

`images`は1件以上を必須とし、同じImage IDを重複させない。
各Imageの`items`は必須だが空配列を許可する。

`label_id`は省略または`null`を許可する。
指定する場合はURLのProjectに属するLabelDefinitionを参照しなければならない。
Source Label名の一致による自動対応は行わない。

Request全体を検証してから1つのDB Transactionで保存する。
1件でも不正なImageまたはPreLabelがある場合は`400 Bad Request`として全件を拒否する。
Image transform不一致またはworkflow制約は`409 Conflict`として全件を拒否する。
どちらの場合も既存のCurrent Pre-Labelsを変更しない。

```jsonc
// Response 400
{
  "error": "invalid pre-label import",
  "path": "images[1].items[3].coordinates",
  "reason": "bbox exceeds normalized image bounds"
}
```

成功したImportはRequestに含まれるImageだけのCurrent Pre-Labelsを置換する。
Requestに含まれないImageと既存Annotationは変更しない。
`items: []`を持つImageは正常な空結果としてCurrent Pre-Labelsを空にする。

このEndpointはモデル、Provider、endpoint、認証情報を受け取らず、外部HTTP通信を行わない。

#### Get Current Pre-Labels

```jsonc
// GET /images/:imageId/pre-labels
// Response 200
{
  "import": {
    "id": "0198...",
    "project_id": "0193...",
    "source": {
      "name": "layout-detector",
      "version": "2026-07-24.1",
      "reference": "batch-2026-07-24"
    },
    "imported_at": "2026-07-24T12:00:00Z"
  },
  "coordinate_space": {
    "width": 2480,
    "height": 3508,
    "rotation": 0,
    "flip_h": false,
    "flip_v": false
  },
  "items": [
    {
      "id": "0199...",
      "import_id": "0198...",
      "image_id": "0194...",
      "external_id": "detection-42",
      "type": "bbox",
      "coordinates": {
        "x": 0.10,
        "y": 0.20,
        "width": 0.30,
        "height": 0.05
      },
      "confidence": 0.94,
      "source_label": {
        "key": "0",
        "name": "header"
      },
      "label_id": "0195...",
      "state": "pending",
      "accepted_annotation_id": null,
      "decided_at": null
    }
  ]
}
```

PreLabelはIDの昇順で返す。
現在のtransformに一致するImportがない場合は`import: null`、`coordinate_space: null`、`items: []`を返す。
以前のImportに属するPreLabelは判断履歴として保持するが、このEndpointの操作対象には含めない。

### Guidelines

| Method | Path | Description | Phase |
|--------|------|-------------|-------|
| `POST` | `/projects/:projectId/guidelines` | ガイドライン作成 | 4 |
| `GET` | `/projects/:projectId/guidelines` | ガイドライン一覧 | 4 |
| `GET` | `/projects/:projectId/guidelines/:guidelineId` | ガイドライン詳細 | 4 |
| `PATCH` | `/projects/:projectId/guidelines/:guidelineId` | ガイドライン更新 | 4 |
| `DELETE` | `/projects/:projectId/guidelines/:guidelineId` | ガイドライン削除 | 4 |

```jsonc
// POST /projects/:projectId/guidelines
// Request
{
  "title": "BBox Annotation Rules",
  "body": "## Rules\n\n- Include margins...",
  "display_order": 1
}

// Response 201
{
  "id": "0194...",
  "project_id": "0194...",
  "title": "BBox Annotation Rules",
  "body": "## Rules\n\n- Include margins...",
  "display_order": 1,
  "updated_at": "2026-03-23T12:00:00Z"
}
```

`title`は前後の空白を除いた後に1文字以上、`display_order`は0以上の整数を必須とする。
`body`は空文字列を許可し、Markdown原文を保存する。
一覧は`display_order`、`title`、Guideline IDの順で返し、GuidelineがないProjectでは`items: []`を返す。
取得、更新、削除では、Guideline IDが存在しない場合と指定Projectに属さない場合のどちらも`404 Not Found`を返す。
更新は`title`、`body`、`display_order`の全項目を必須とし、成功時に`updated_at`を更新する。

### Comments

| Method | Path | Description | Phase |
|--------|------|-------------|-------|
| `POST` | `/images/:imageId/comments` | コメント作成 | 4 |
| `GET` | `/images/:imageId/comments` | コメント一覧 | 4 |
| `PATCH` | `/images/:imageId/comments/:commentId` | 解決状態の更新 | 4 |
| `DELETE` | `/images/:imageId/comments/:commentId` | コメント削除 | 4 |

```jsonc
// POST /images/:imageId/comments
// Request
{
  "author": "reviewer-1",
  "body": "This bbox should include the header margin",
  "type": "issue",
  "annotation_id": "0194..."
}

// Response 201
{
  "id": "0194...",
  "author": "reviewer-1",
  "body": "This bbox should include the header margin",
  "type": "issue",
  "annotation_id": "0194...",
  "image_id": "0194...",
  "resolved": false,
  "target_deleted": false,
  "created_at": "2026-03-23T12:00:00Z",
  "updated_at": "2026-03-23T12:00:00Z"
}

// PATCH /images/:imageId/comments/:commentId
// Request
{
  "resolved": true
}
```

`annotation_id`は省略または`null`の場合にImage全体を対象とし、指定した場合は同じImageに属する永続化済みAnnotationを対象とする。
存在しないAnnotation、別Imageまたは別ProjectのAnnotationを指定した場合は`404 Not Found`を返す。
`author`は認証連携前の表示名として前後の空白を除いた1文字以上を必須とし、作成後は変更しない。
`body`は空白だけの値を拒否し、Markdown原文を保存する。
`type`は`question`、`issue`、`note`のいずれかを必須とする。
`resolved`は作成時に`false`となり、`PATCH`ではbooleanの`resolved`だけを必須とする。
一覧は`created_at`、Comment IDの順で返し、CommentがないImageでは`items: []`を返す。
更新と削除では、Comment IDが存在しない場合と指定Imageに属さない場合のどちらも`404 Not Found`を返す。
Annotationを削除してもCommentは削除せず、以後の一覧では`annotation_id`を保持したまま`target_deleted: true`を返す。
Imageを削除した場合は、そのImageに属するCommentも削除する。
Comment本文の表示ではraw HTMLと埋め込み画像を描画しない。

### Export

| Method | Path | Description | Phase |
|--------|------|-------------|-------|
| `GET` | `/projects/:projectId/export?format=json\|coco\|yolo` | プロジェクト全体エクスポート | 1, 3 |
| `GET` | `/images/:imageId/export?format=json` | 画像単位GOAT JSONエクスポート | 1 |

#### Supported Formats

| Format | `format` param | 内容 | Phase |
|--------|---------------|------|-------|
| **GOAT JSON** | `json` | 独自JSON形式。全情報を完全に保持（エッジ含む） | 1 |
| **COCO** | `coco` | COCO Object Detection format | 3 |
| **YOLO** | `yolo` | YOLO txt format | 3 |

`format`を省略した場合は`json`として扱う。
未対応の`format`、または画像単位APIへの`coco`と`yolo`の指定は`400 Bad Request`を返す。

COCOとYOLOはProject単位のZIPを返す。
どちらも保存中の原画像を同梱し、変換後の正規化Annotation座標を原画像の座標空間へ戻してから出力する。
変換は表示行列の逆順で`rotation^-1 → flip_v^-1 → flip_h^-1`を各点へ適用する。
BBoxは4隅を変換した後の外接矩形、Polygonは各頂点を変換した結果を使用する。

COCOは全Label categoryのBBoxとPolygonを対象にし、category IDを1から割り当てる。
YOLOは`object` categoryのBBoxだけを対象にし、class IDを0から割り当てる。
Labelは名前、Label IDの順で安定ソートしてIDを決める。
COCOの`categories[].goat_label_id`とYOLOの`classes.json`により、変換後のclass IDからLabel IDを復元できる。

YOLOで対象外となるPolygonと`object`以外のAnnotationは`manifest.json`の`warnings`へAnnotation IDと理由を記録する。
座標Schemaが不正なAnnotation、LabelのないAnnotation、Project外のLabelを参照するAnnotationは変換不能として`422 Unprocessable Entity`を返し、ZIPを生成しない。
空のProjectは空の画像一覧とAnnotation一覧、定義済みのクラス対応を持つ有効なZIPとして返す。

COCO/YOLOはノード（Annotation）のみ対応する。
エッジ（Reading Order、KV等）のグラフ構造を収録しないことは、各ZIPの`manifest.json`へ機械可読な値として記録する。
`manifest.json`の`images`はImage ID、元ファイル名、ZIP内path、原画像サイズ、rotation、flipを保持し、ID化した画像名から元のImage情報を復元できるようにする。

#### COCO ZIP

```text
project-coco.zip/
├── images/default/<image_id>.<ext>
├── annotations/instances_default.json
└── manifest.json
```

COCOの`images`は原画像の`width`と`height`を持つ。
BBoxはピクセル単位の`bbox: [x, y, width, height]`として出力する。
Polygonはピクセル単位の`segmentation`、外接`bbox`、shoelace formulaで計算した`area`として出力する。

#### YOLO ZIP

```text
project-yolo.zip/
├── images/train/<image_id>.<ext>
├── labels/train/<image_id>.txt
├── data.yaml
├── classes.json
└── manifest.json
```

YOLOの各行は`class_id center_x center_y width height`であり、座標は原画像空間の0から1までの正規化値とする。
Annotationのない画像にも空のLabel fileを作成し、画像とAnnotation fileを1対1で対応させる。
Project全体を`train`へ配置し、train/validationの自動分割は行わない。

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
