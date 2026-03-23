# ADR-0003: Canvas描画ライブラリ

## Status

Accepted

## Context

アノテーション（BBox、Polygon等）の描画・操作を行うCanvas描画ライブラリを選定する。
React上で動作し、Pan/Zoom、図形の移動・リサイズ、イベントハンドリングが必要。

## Options

### Konva (react-konva)

- シーングラフベースの2D Canvas描画ライブラリ
- react-konva公式でReactとの統合が自然
- BBox/Polygon等の図形描画、移動・リサイズが組み込み
- Pan/ZoomはStageレベルで対応
- 学習コストが低〜中

### Fabric.js

- オブジェクトモデルベースの最も多機能なCanvas操作ライブラリ
- 図形操作機能が最も充実
- Reactとの統合はラッパー経由で薄い（命令的操作が主体）
- v6で大幅リファクタリング中（破壊的変更あり、安定性に懸念）
- 大量図形でパフォーマンスが低下する傾向

### Canvas API直接

- 最も自由度が高い
- Pan/Zoom、図形操作、ヒットテスト等すべて自前実装が必要
- 実装コストが非常に高い
- React統合はuseRef + 命令的操作

### OpenLayers

- 地図表示ライブラリだがCVATが採用している実績あり
- Pan/Zoom（タイル表示）が本領
- 地図概念（投影法、座標系等）の理解が必要で学習コストが高い
- アノテーションツールとしてはオーバースペック

## Decision

**Konva (react-konva)** を採用する。

## Rationale

- React統合が最も自然で、**宣言的にアノテーションUI**を構築できる
- BBox/Polygon描画、移動・リサイズ、Pan/Zoomといった**必要機能が組み込み**
- **学習コストが低く**、Phase 1の素早い立ち上げに適する
- Fabric.jsのv6不安定リスクを回避
- Canvas API直接の実装コストを回避
- メンテナンスが活発で長期的なリスクが低い
