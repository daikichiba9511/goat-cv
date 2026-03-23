# ADR-0002: Database & Query

## Status

Accepted

## Context

データ永続化のためのDB・クエリ手法を選定する。
Phase 1は単一ユーザー。Phase 4で共同編集対応時にPostgreSQLへの移行を見据える。

## Options

### SQLite + sqlc

- DBはファイル1つ、外部プロセス不要で最小構成
- sqlcはSQLからGo型を自動生成し、型安全にクエリ実行
- 生SQLを書くため発行クエリが透明
- PostgreSQL移行時もSQLの知識・資産がそのまま活きる
- マイグレーションツール（goose等）は別途必要

### SQLite + GORM

- Go構造体ベースでDB操作、SQL知識が薄くても使える
- AutoMigrateで手軽にスキーマ管理
- リフレクションベースのため型安全性が低い
- N+1等のパフォーマンス問題が隠れやすい

### PostgreSQL + sqlc

- 初手から本番想定のDB
- 同時接続に強く、共同編集フェーズで移行不要
- ローカル開発にDocker等が必要でセットアップが重い

## Decision

**SQLite + sqlc** を採用する。Phase 4でPostgreSQLへ移行する。

## Rationale

- Phase 1は単一ユーザーなので**SQLiteのシンプルさが最適**
- sqlcの「SQL → Go型生成」は**パフォーマンスの透明性**と**型安全性**を両立
- 生SQLを書く資産はPostgreSQL移行時にそのまま流用可能
- GORMのリフレクションベースの暗黙的挙動はデバッグコストを生む
- PostgreSQLの初手導入はPhase 1には過剰
