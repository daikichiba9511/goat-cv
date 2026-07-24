# ADR-0006: Pre-Inference境界

## Status

Superseded by [ADR-0007](0007-pre-label-import-boundary.md)

## Context

外部モデルの出力をAnnotation候補として扱うには、Provider固有形式、CandidateとAnnotationの境界、再実行、失敗、採用時の保存単位を決める必要がある。
GOATはローカル実行とImage単位の同期Graph保存を現在の前提としている。

## Options

### Provider固有APIを直接公開する

FrontendまたはUsecaseがProviderごとのRequestとResponseを扱う。
導入は短いが、Providerを増やすたびにDomainとUIへ固有fieldが広がる。

### CandidateをClientだけに保持する

Provider応答をCanvasへ直接表示し、採用時に通常のAnnotationとして保存する。
永続Schemaは増えないが、reload、重複要求、model version、採用由来を追跡できない。

### Provider Adapterと永続Candidateを使う

Adapterがベンダー応答を正規化し、RunとCandidateをServerへ保存する。
Candidate判断は既存のImage Graph保存Transactionへ含める。

### 非同期Job Queueを先に導入する

Runを非同期Jobとして実行し、進捗をpollingまたはpushで通知する。
長時間推論には適するが、Worker運用、再配送、cancel、進捗管理を最初から必要とする。

## Decision

Provider Adapterと永続Candidateを採用する。
最初の実行方式はtimeout付きの同期HTTP呼び出しとし、Runの状態をDBへ記録する。

ProviderにはBackendがtransformを適用した画像binaryを渡し、BBoxとPolygonを変換後画像に対する正規化座標で受け取る。
Candidateは利用者が採用するまでAnnotationにせず、採用、修正、破棄をImage Graph保存と同じTransactionで確定する。
再実行が成功した場合は新しいRunだけを操作対象にするが、以前に採用したAnnotationとRun履歴は残す。

当時の詳細なSchemaと状態遷移は[PR #46](https://github.com/daikichiba9511/goat-cv/pull/46)の履歴に残す。
現在の正本にはこのADRではなく[ADR-0007](0007-pre-label-import-boundary.md)を使用する。

## Rationale

- ベンダー固有の認証、座標、Label形式をAdapter内へ閉じ込められる
- Candidateと確定Annotationを永続Schemaでも区別できる
- 既存Graph保存とCandidate判断の部分Commitを防げる
- transform fingerprintによりProvider出力とCanvas座標の一致を検証できる
- 永続Runを保ったまま、計測後に同期実行を非同期Workerへ置き換えられる

## Consequences

- Run、Candidate、Label MappingのtableとRepositoryが必要になる
- Backendで回転と反転を適用した推論用画像を生成する必要がある
- Image Graph保存RepositoryがCandidate判断も所有する
- Provider設定にはtimeout、最大応答byte数、最大Candidate件数が必要になる
- 非同期batch、進捗、cancelは別の判断と実装を要する
