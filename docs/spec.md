# GOAT Product Specification

GOAT は **Go CV Annotation Tool** の略称であり、画像データセット作成のための Computer Vision アノテーションツールである。
帳票・フォーム・請求書などのドキュメント画像は初期の重点ユースケースだが、ツール自体はドキュメント画像に限定しない。

## Goals

- 画像に対して BBox、Polygon、Annotation 間の Edge を付与できる
- Object Detection、Reading Order、Table Analysis、Information Extraction、KV Extraction のデータセット作成を支援する
- まず単一ユーザー・同期保存のローカルツールとして小さく成立させる
- 将来的に QA、事前推論、非同期共同編集へ拡張できるデータモデルとUI構造を保つ

## Non-Goals

- Phase 1-4 ではユーザー認証・権限管理を実装しない
- Phase 1-4 ではクラウドストレージ、ジョブキュー、マルチテナント運用を前提にしない
- 任意角度回転、skew補正、OCR前処理はアノテーション前の外部処理に任せる
- CVAT 互換APIやCVATプロジェクトインポートは初期スコープに含めない

## Target Users

| User | Needs |
|------|-------|
| Annotator | 画像を選び、ラベル付き領域と関係を素早く付与する |
| Reviewer | アノテーションを確認し、差戻し・承認・コメントを行う |
| Dataset owner | プロジェクト、ラベル体系、ガイドライン、エクスポート形式を管理する |
| ML engineer | 学習に使える JSON、COCO、YOLO 形式のデータを取得する |

## Core Concepts

| Concept | Description |
|---------|-------------|
| Project | 画像、ラベル、ガイドライン、エクスポートの単位 |
| Image | アノテーション対象の画像 |
| LabelDefinition | プロジェクト単位のラベル定義。カテゴリで用途を分ける |
| Annotation | 画像上の領域。BBox または Polygon で表す |
| Edge | Annotation 間の有向関係。Reading Order、KV、Table Cell を表す |
| Guideline | プロジェクト単位の作業マニュアル |
| Comment | Image または Annotation に紐づく QA・エスカレーション記録 |

## Functional Requirements

### Project and Label Management

- Project を作成、一覧表示、更新、削除できる
- Project ごとに LabelDefinition を作成、一覧表示、更新、削除できる
- LabelDefinition は `object`、`entity`、`key`、`value`、`table`、`cell` のカテゴリを持つ
- Canvas 上の Annotation は選択した LabelDefinition の色で表示される

### Image Management

- Project に画像をアップロードできる
- アップロードした画像を一覧表示し、選択できる
- 画像メタデータとして原画像サイズ、変換後サイズ、回転、反転、ステータスを保持する
- 画像ファイルはローカルファイルシステムに保存し、APIから配信する
- 回転は `0`、`90`、`180`、`270` のみをサポートする

### Annotation Editing

- BBox Annotation を作成、選択、移動、リサイズ、削除できる
- Polygon Annotation を作成、選択、編集、削除できる
- Polygonは3個以上の頂点で確定し、確定前は輪郭と頂点をプレビューする
- 3個未満の頂点ではPolygonを確定できず、Cancelした作成途中の頂点は保存対象に含めない
- 選択したPolygonは頂点handleを表示し、各頂点を画像範囲内かつ他の頂点と重ならない位置へ移動できる
- Polygonの頂点順序と選択中のLabelは保存と再読込の後も維持する
- Annotation は正規化座標で保存する
- Zoom/Pan は表示操作であり、保存座標には影響しない
- 一括保存により、Image 単位の Annotation と Edge を1回の操作で置き換えられる
- 保存に失敗した場合、編集中の Annotation と Edge を保持し、理由を表示して再試行できる
- Annotation InspectorでLabel、図形種別、接続中のEdge数を一覧できる
- Annotation InspectorはLabelと図形種別で表示を絞り込み、Canvasと選択状態を同期できる
- Annotation InspectorからAnnotationを削除すると、接続中のEdgeもローカル編集状態から除去する

### Graph Annotation

- Annotation 間に Edge を作成、表示、削除できる
- Edge type は `reading_order`、`key_value`、`table_cell` をサポートする
- Edge は同一 Image 内の Annotation 間にのみ作成できる
- 同一 `(source, target, type)` の重複 Edge は作成できない
- `reading_order` は DAG とし、閉路を作成できない
- `key_value` は Key category から Value category への 1:1 関係とする
- `table_cell` は Table category から Cell category への 1:N 関係とする
- Edge 作成時は relation type と source、target の Label category 条件を確認できる
- 条件違反、重複、多重度違反、循環を作る操作は編集グラフへ追加せず、拒否理由を表示する
- relation type を切り替えた場合は、選択途中の source を破棄する
- Edge type は Canvas 上の色、線種、短い名称で区別する

### Workflow and QA

- Image status は `pending`、`annotated`、`in_review`、`approved`、`rejected`、`escalated` を持つ
- Annotator は作業完了時に `annotated` へ変更できる
- Reviewer は `in_review`、`approved`、`rejected` へ変更できる
- 判断に迷う画像または Annotation に対して `escalated` を設定できる
- Project に複数の Guideline を作成、表示、更新、削除できる
- Guideline は `title`、Markdown形式の `body`、0以上の `display_order` を持つ
- Guideline は `display_order`、`title`、Guideline IDの順に表示し、同順位でも順序を安定させる
- AnnotatorのGuideline panelを開閉しても、Canvasの選択中Annotationと未保存の編集状態を維持する
- Guidelineのraw HTMLと埋め込み画像は描画せず、Markdownからscriptを実行できないようにする
- Image全体または永続化済みAnnotationにCommentを作成し、削除できる
- Commentは`question`、`issue`、`note`のtype、Markdown形式のbody、author、resolvedを持つ
- Commentのresolvedは解決済みと未解決の間で相互に変更できる
- Annotationを選択した場合は、そのAnnotationに属するCommentをComments panelへ表示する
- Annotationを削除した場合は属するCommentも削除し、同じAnnotation IDの編集とgraph保存では保持する
- Commentのraw HTMLと埋め込み画像は描画せず、Markdownからscriptを実行できないようにする

### Export

- GOAT JSON は Project または Image 単位でエクスポートできる
- GOAT JSON は Annotation、Edge、LabelDefinition、Image transform 情報を保持する
- COCO export は全 Label category の BBox と Polygon を対象にする
- YOLO export は `object` category の BBox だけを対象にする
- COCO と YOLO は Project 単位の ZIP として、原画像、Annotation、クラスと Label ID の対応、Export 制約を取得できる
- COCO と YOLO は原画像をそのまま同梱し、変換後の正規化座標を原画像の座標空間へ逆変換する
- YOLO で対象外となる有効な Annotation は `manifest.json` の警告へ記録し、黙って欠落させない
- 座標 Schema が不正な Annotation、Label のない Annotation、別 Project の Label を参照する Annotation が1件でもあれば COCO/YOLO export 全体を失敗させる
- Edge を含むグラフ構造は GOAT JSON のみで完全に保持する

### Pre-Inference

- 外部モデルAPIへ画像を渡し、推論候補を取得できる
- 推論候補は確定 Annotation とは区別して表示する
- ユーザーは候補を採用、修正、破棄できる

### Collaboration

- 複数ユーザーが同じ Image を編集できる方向へ拡張する
- Phase 6 では最初に同期方式の技術検証を行う
- WebSocket 差分同期、CRDT、OT のいずれを採用するかは検証結果で決める

## Non-Functional Requirements

| Category | Requirement |
|----------|-------------|
| Usability | Annotator は Canvas から視線を大きく外さずにラベル選択、保存、画像切替を行える |
| Data integrity | Annotation と Edge は Image 単位の1 Transactionで保存する。検証またはDB処理に失敗した場合は両方を変更しない |
| Portability | Phase 1-4 はローカル開発環境で起動できる |
| Performance | 1画像あたり数百 Annotation までは操作が破綻しない |
| Extensibility | Repository、Usecase、Handler の層を分け、SQLite から PostgreSQL へ移行できる余地を残す |
| Recoverability | 画像ファイルとSQLite DBは通常のファイルバックアップで退避できる |

## Data and Coordinate Rules

- Annotation 座標は変換後の画像空間に対する正規化値として保存する
- BBox は有限値の `x`、`y`、`width`、`height` を必須とし、正の面積を持つ矩形全体が `0.0` から `1.0` に収まるものだけを保存する
- Polygon は有限値の `x`、`y` を持つ相異なる点を3個以上必須とし、各点が `0.0` から `1.0` に収まるものだけを保存する
- 一括保存に不正な Annotation が1件でも含まれる場合、API はリクエスト全体を拒否し、既存 Annotation を変更しない
- 表示行列は原画像の中心を基準に `flip_h -> flip_v -> rotation` の順で合成する
- 回転・反転を変更した後の既存 Annotation は原則として再確認が必要である
- API は JSON を基本とし、画像アップロードのみ `multipart/form-data` を使う

## Phase 1 Acceptance Criteria

- Project を作成できる
- LabelDefinition を作成し、BBox 作成時に割り当てられる
- 画像をアップロードし、Canvas に表示できる
- BBox を作成、移動、リサイズ、削除できる
- Annotation を保存し、画面再読み込み後に復元できる
- GOAT JSON をエクスポートできる
- Backend の `go test ./...` が成功する
- Frontend の `npm run build` が成功する

## Open Questions

- Phase 6 の共同編集方式は WebSocket 差分同期、CRDT、OT のどれを採用するか
- Reviewer と Annotator の権限をいつ導入するか
- Pre-Inference のモデルAPI形式を GOAT 固有にするか、設定で複数形式を許容するか
