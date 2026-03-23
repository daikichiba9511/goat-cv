# ADR-0004: State Management

## Status

Accepted

## Context

フロントエンドの状態管理ライブラリを選定する。
アノテーションツール特有の要件として以下がある：

- undo/redo対応
- 将来のWebSocketによる共同編集との連携
- 一度導入すると剥がしにくいため、負債化リスクの低さが重要

## Options

### Zustand

- 単一Storeの薄いFlux系ライブラリ
- コンポーネントへの影響は`useStore(selector)`のみで侵食度が低い
- React外からもStoreにアクセス可能（テスト、WebSocketハンドラ等）
- `temporal`ミドルウェアでundo/redo対応
- `subscribe`でWebSocket同期が自然に実装可能

### Jotai

- atomベースのボトムアップ状態管理
- atom単位の細かい再レンダリング最適化が自然
- atom/useAtomがコンポーネント全体に浸透し侵食度が高い
- 共同編集時の一括同期でatom個別管理が複雑になる懸念
- undo/redoは全atomを横断したスナップショットが必要で複雑

### Redux Toolkit

- 最も成熟したFlux系ライブラリ
- DevToolsが最も充実
- sliceの定義等ボイラープレートが多い
- 小さく始めるには過剰

### useReducer + Context

- React標準API、ライブラリ依存ゼロで最もロックインが低い
- Context分割しないと全再レンダリングが発生する
- ミドルウェア（undo/redo、永続化等）を自前実装する負担が大きい

## Decision

**Zustand** を採用する。

## Rationale

- **侵食度が最も低い**: Store定義が局所的で、コンポーネントは`useStore(s => s.xxx)`のみ
- **剥がしやすい**: Storeは普通のオブジェクトであり、将来別の仕組みに移行しても影響範囲が小さい
- **将来要件に強い**: undo/redo(`temporal`)、WebSocket同期(`subscribe`)、immutable更新(`immer`)がミドルウェアとして後付け可能
- **React外アクセス**: WebSocketハンドラやテストからStore操作できるため、共同編集対応の布石になる
- Jotaiのatom浸透による剥がしにくさのリスクを回避
