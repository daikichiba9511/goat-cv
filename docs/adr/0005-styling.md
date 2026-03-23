# ADR-0005: Styling

## Status

Accepted

## Context

フロントエンドのスタイリング手法を選定する。
Phase 1ではUIのこだわりは不要で、素早く構築できることを優先する。

## Options

### Tailwind CSS

- ユーティリティファーストCSS
- HTMLにクラスを直接記述、CSSファイル不要
- purgeにより未使用スタイルが自動除去
- 既製UIコンポーネントはなく自前構築

### CSS Modules

- CSS標準に最も近い、ライブラリ依存なし
- スコープ付きCSSでスタイル衝突を防止
- 既製コンポーネントなし

### shadcn/ui + Tailwind

- Tailwindベースの既製UIコンポーネント集
- ダイアログ、ボタン等がすぐ使える
- UIにこだわる場合に有効だが、現時点では過剰

## Decision

**Tailwind CSS** を素のまま採用する。shadcn/ui等のコンポーネントライブラリは必要になった時点で足す。

## Rationale

- **UIのこだわりは不要**という方針に合致
- Tailwind単体で十分素早くレイアウトが組める
- 依存を最小限にしてフロントエンドの構成をシンプルに保つ
- shadcn/uiは後から追加可能なので、現時点では導入しない
