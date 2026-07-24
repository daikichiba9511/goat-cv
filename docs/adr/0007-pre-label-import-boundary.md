# ADR-0007: Pre-Label Import境界

## Status

Accepted

## Context

外部モデルの推論結果をアノテーション作業へ利用したい。
ただし、GOATはComputer Visionアノテーションツールであり、モデル実行基盤ではない。

[ADR-0006](0006-pre-inference-boundary.md)は、GOATが外部Providerへ画像を送り、推論を実行する前提を採用していた。
この前提ではGOATがendpoint、認証、timeout、retryを管理する。
画像変換と実行状態もGOATの責務になり、製品の範囲を越える。

[CVATのTask操作](https://docs.cvat.ai/docs/manual/basics/create-annotation-task/)には、Upload annotationとAutomatic Annotationがある。
両者は別の操作である。
GOATでは前者に近いImport境界を採用し、推論結果を確認前の事前ラベルとして扱う。

## Options

### GOATが外部モデルAPIを呼び出す

GOATが画像を送信し、Provider応答を正規化する。
操作は一画面で完結するが、推論基盤の設定と障害処理がBackendへ入る。

### 外部結果を通常のAnnotationとしてImportする

既存のGraphへ直接追加する。
Schemaは単純だが、確認前の推論結果と利用者が確定したAnnotationを区別できない。

### 生成済み結果をPreLabelとしてImportする

外部PipelineがGOAT固有JSONを生成し、GOATは検証済みの結果をPreLabelとして保存する。
利用者の判断後だけ通常のAnnotationへ変換する。

## Decision

生成済み結果をPreLabelとしてImportする。
[ADR-0006](0006-pre-inference-boundary.md)のProvider AdapterとInference Runは採用しない。

Import単位はProjectとし、同じProjectに属する複数Imageの結果を1つのJSONで受け付ける。
1件でも不正な入力があれば全件を拒否し、部分Importを行わない。

PreLabelはSource Label、任意のconfidence、任意のGOAT Label ID、取り込み時のcoordinate spaceを保持する。
採用、修正、破棄はFrontendでstageし、Image Graph保存と同じTransactionで確定する。

詳細なSchemaと振る舞いは[Pre-Label Import Specification](../pre-label-import.md)を正本とする。

## Rationale

- 推論手段をPython script、Notebook、batch service、任意のModel Serverから独立させられる
- GOATにProvider認証と実行障害の責務を持ち込まずに済む
- 推論結果と確定Annotationを永続Schemaでも区別できる
- Project単位のFileで複数Imageの結果を一度に取り込める
- Graph保存とPreLabel判断の部分Commitを防げる

## Consequences

- 外部処理はGOAT Pre-Label JSON 1.0を生成する必要がある
- GOATはPreLabelImport、PreLabelImportImage、PreLabelを永続化する
- Vendor固有形式を使う場合はGOATの外で変換する
- Import時と判断時にImage transformとworkflowを検証する
- モデル実行、Provider設定、timeout、retry、job管理はGOATへ追加しない
