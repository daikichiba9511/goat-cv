# Phase 1: Single User, Sync

画像アップロード、BBox描画・保存・読込の最小構成。

## Backend

### Project Setup
- [x] Go module 初期化 (`go mod init`)
- [x] Chi, CORS, UUID, SQLite 依存追加
- [x] ディレクトリ構成作成 (cmd/, internal/, db/)
- [x] SQLite 接続ヘルパー + WAL mode + foreign keys
- [x] main.go エントリポイント (Chi router, CORS, health endpoint)
- [x] マイグレーション実行機能

### Database
- [x] マイグレーション (001_init.sql): projects, images, annotations, label_definitions, edges
- [x] sqlc 設定 (sqlc.yaml) + クエリ定義 (project, label, image, annotation, edge)

### Domain
- [x] Project ドメインモデル
- [x] Image ドメインモデル (original_width/height, width/height, rotation, flip_h, flip_v, status) + EffectiveDimensions
- [x] Annotation ドメインモデル (type, coordinates JSON, label_id) + BBox/Polygon座標型
- [x] LabelDefinition ドメインモデル (name, color, category)
- [x] Edge ドメインモデル (source, target, type)

### Repository (SQLite)
- [x] convert.go (sqlcgen型 → Domain型変換)
- [x] ProjectRepository (CRUD)
- [x] ImageRepository (CRUD + Transform更新 + Status更新)
- [x] AnnotationRepository (CRUD + BulkReplace)
- [x] LabelDefinitionRepository (CRUD)
- [x] EdgeRepository (CRUD + BulkReplace)

### Usecase
- [x] Project ユースケース (CRUD)
- [x] Image ユースケース (アップロード + メタデータ抽出 + Transform更新)
- [x] Annotation ユースケース (CRUD + BulkReplace)
- [x] LabelDefinition ユースケース (CRUD)

### Handler (API)
- [x] Router 設定 (Chi) + main.go 全ルート接続
- [x] POST/GET/PATCH/DELETE `/projects`
- [x] POST/GET/DELETE `/projects/:projectId/images`
- [x] GET `/images/:imageId` (メタ情報)
- [x] GET `/images/:imageId/file` (画像配信)
- [x] PATCH `/images/:imageId` (Transform更新)
- [x] POST/GET/PATCH/DELETE `/images/:imageId/annotations`
- [x] PUT `/images/:imageId/annotations` (一括保存)
- [x] POST/GET/PATCH/DELETE `/projects/:projectId/labels`
- [x] GET `/projects/:projectId/export` (GOAT JSON)
- [x] GET `/images/:imageId/export` (GOAT JSON)
- [x] CORS ミドルウェア (開発用)

### Image Handling
- [x] multipart アップロード処理
- [x] 画像ファイルのローカルFS保存
- [x] 画像サイズ (width/height) の自動取得
- [x] storage/ ディレクトリの .gitignore 設定

## Frontend

### Project Setup
- [x] Vite + React + TypeScript 初期化
- [x] Tailwind CSS 設定 (@tailwindcss/vite)
- [x] react-konva, zustand, react-router-dom 依存追加
- [x] Vite proxy 設定 (API → backend :8080)

### Types
- [x] Project 型定義
- [x] Image 型定義 (transform fields 含む)
- [x] Annotation 型定義 (BBox coordinates)
- [x] LabelDefinition 型定義

### API Client
- [x] Project API (CRUD)
- [x] Image API (upload, list, get, file URL)
- [x] Annotation API (CRUD + bulk save)
- [x] LabelDefinition API (CRUD)

### Stores (Zustand)
- [x] projectStore (current project, project list, labels, images)
- [x] annotationStore (annotations, selected, dirty flag, addBBox, save)

### Pages
- [x] ProjectList — プロジェクト一覧・作成
- [x] Annotator — メインアノテーション画面
- [x] React Router 設定

### Canvas (Konva)
- [x] AnnotationCanvas — 画像表示 + アノテーション描画
- [x] 画像の読み込みと表示 (rotation/flip 適用)
- [x] Pan (Stage draggable)
- [x] Zoom (wheel, pointer中心, 0.1x-10x)
- [x] BBoxTool — BBox 描画 (mousedown → mousemove → mouseup)
- [x] BBox 選択・移動・リサイズ (Transformer)
- [x] BBox 削除 (Delete/Backspace)
- [x] ラベル割り当て (activeLabel → 描画時に適用)
- [x] ラベル色でBBox描画色を変える

### Sidebar
- [x] 画像一覧 (ファイル名リスト)
- [x] 画像選択で Canvas 切り替え
- [x] 画像アップロードボタン

### Toolbar
- [x] ツール切り替え (Select / BBox / Pan)
- [x] 保存ボタン (一括保存 PUT, dirty表示)
- [x] 回転ボタン (0° → 90° → 180° → 270°)
- [x] 反転ボタン (flip_h, flip_v)

### Label Management
- [x] ラベル一覧表示 (Right Panel)
- [x] ラベル作成 (name, color picker, category select)
- [x] ラベル選択 → 次の描画に適用

## Integration
- [x] Backend 起動 → Frontend からAPI呼び出しの動作確認
- [x] 画像アップロード → Canvas表示 → BBox描画 → 保存 → リロード後に復元の E2E 動作確認
- [x] GOAT JSON エクスポートの動作確認

## Review

Phase 1 完了。全タスク実装済み・動作確認済み。

### 実装済み機能
- Backend: Go + Chi + SQLite + sqlc. 全CRUD API + Export API
- Frontend: React + Vite + Konva + Zustand + Tailwind
- プロジェクト管理、画像アップロード、BBox描画/選択/移動/リサイズ/削除
- ラベル管理（作成/選択/色反映）、一括保存、Zoom/Pan
- Rotation/Flip (表示 + API永続化)、GOAT JSON エクスポート
