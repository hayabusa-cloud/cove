[![Go Reference](https://pkg.go.dev/badge/code.hybscloud.com/cove.svg)](https://pkg.go.dev/code.hybscloud.com/cove)
[![Go Report Card](https://goreportcard.com/badge/github.com/hayabusa-cloud/cove)](https://goreportcard.com/report/github.com/hayabusa-cloud/cove)
[![Coverage Status](https://codecov.io/gh/hayabusa-cloud/cove/graph/badge.svg)](https://codecov.io/gh/hayabusa-cloud/cove)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

[English](README.md) | [简体中文](README.zh-CN.md) | [Español](README.es.md) | **日本語** | [Français](README.fr.md)

# cove

Go における [kont](https://code.hybscloud.com/kont) サスペンションのための余代数的コンテキスト層であり、kont の代数的エフェクトの余効果双対。

## 概要

外部ランタイム、たとえばプロアクター、イベントループ、ディスパッチャが kont サスペンションを一つずつ処理するとき、各中断された操作には操作自体に含まれない状態が必要になることがある。ディスパッチ予算、リングケイパビリティ、プロトコルフェーズ、バッファグループの有効性など。明示的なコンテキストキャリアがなければ、こうした状態はその場しのぎの副マップや暗黙のグローバル変数に置かれ、合成が壊れる。

cove は環境コンテキストを値やサスペンションと対にすることで、コンテキストが非同期ステッピング境界を型付きで合成可能なデータとして通過できるようにする。

```go
_, sv := cove.StepExprWith(Runtime{Budget: 8}, computation)
for sv.Suspension != nil {
    result := dispatch(sv.Ask(), sv.Op())
    _, sv = sv.ResumeWith(result, func(r Runtime) Runtime {
        r.Budget--
        return r
    })
}
```

パッケージはポリシーフリーである。コンテキストの保持と検査のみを行い、スケジューリング・リトライ・ディスパッチは行わない。

## インストール

```bash
go get code.hybscloud.com/cove
```

Go 1.26+ が必要。

## コア型

| 型 | 用途 |
|----|------|
| `View[C, A]` | 余モナドキャリア：環境コンテキスト C の下の値 A |
| `SuspensionView[C, A]` | 環境コンテキストと対になった kont サスペンション |
| `Req[C]` | C 上の反変述語関手（クロージャ形式）：`func(C) bool` |
| `ReqExpr[C]` | C 上の反変述語関手（脱関数化形式）|
| `Rule[C]` / `RuleExpr[C]` | 診断用 `Report` 付きの名前付き述語 |
| `Checked[C, A]` / `CheckedExpr[C, A]` | 要件でゲートされた値 |
| `Guarded[C, A]` / `GuardedExpr[C, A]` | 名前付きルールでゲートされた値 |

`Ambient` はコンテキスト型パラメータ (C, D) を制約する。`Focus` は値型パラメータ (A, B) を制約する。

## 文脈付きステッピング

`StepWith` と `StepExprWith` は kont 計算を評価し、最初のサスペンションを環境コンテキストと対にする。各サスペンションはアフィンであり、`Resume`、`ResumeWith`、または `Discard` によりちょうど一度だけ消費する。

```go
val, sv := cove.StepExprWith(ctx, expr)
for sv.Suspension != nil {
    op := sv.Op()
    result := handle(op)
    val, sv = sv.Resume(result)
}
```

`ResumeWith` はステップ間でコンテキストを変化させる。たとえば予算を減算し、ケイパビリティを更新し、プロトコルフェーズを進める：

```go
val, sv = sv.ResumeWith(result, func(r Runtime) Runtime {
    r.Budget--
    return r
})
```

`ObserveSuspension` は既存の `kont.Suspension` に要件チェックなしで環境コンテキストを付与する。`CheckSuspension` と `CheckSuspensionExpr` はそのゲート付き版である：

```go
sv, ok := cove.CheckSuspension(runtime, susp, func(r Runtime) bool {
    return r.Budget > 0
})
if !ok {
    susp.Discard()
    return
}
result := dispatch(sv.Ask(), sv.Op())
val, sv = sv.Resume(result)
```

```go
exprReq := cove.ExprAtom(func(r Runtime) bool { return r.Budget > 0 })
sv, ok := cove.CheckSuspensionExpr(runtime, susp, exprReq)
if !ok {
    susp.Discard()
    return
}
result := dispatch(sv.Ask(), sv.Op())
val, sv = sv.Resume(result)
```

`Step` と `StepExpr` はコンテキスト束縛なしで kont の評価器を再エクスポートする。

## 要件

要件はコンテキスト上の述語であり、クロージャ形式とデータ形式の二つがある。どちらも `All`/`Any`/`Not` を備え、`True` と `False` はそれぞれ論理積と論理和の単位元になる。

**クロージャ形式** (`Req`) — 直接的で簡潔：

```go
req := cove.All(
    cove.Req[Runtime](func(r Runtime) bool { return r.Budget > 0 }),
    cove.Req[Runtime](func(r Runtime) bool { return r.CanDispatch }),
)
ok := cove.Need(runtime, req)
```

**データ形式** (`ReqExpr`) — 合成可能なブール構造、合成時のクロージャアロケーションを回避：

```go
expr := cove.ExprAll(
    cove.ExprAtom(func(r Runtime) bool { return r.Budget > 0 }),
    cove.ExprAtom(func(r Runtime) bool { return r.CanDispatch }),
)
ok := cove.NeedExpr(runtime, expr)
```

コンビネータ：`All`/`ExprAll`（論理積 ∧）、`Any`/`ExprAny`（論理和 ∨）、`Not`/`ExprNot`（否定 ¬）、`True`/`ExprTrue`（⊤）、`False`/`ExprFalse`（⊥）。

`Pullback` と `ExprPullback` はコンテキスト射に沿って要件を反変に輸送する。f: C → D が与えられたとき `Pullback(req, f)` は `Req[D]` を `Req[C]` に写す。`ExprPullback` はブール構造を保存し、`ExprNot`、`ExprAll`、`ExprAny` に分配する。

### Req と ReqExpr の使い分け

その場限りの述語には `Req` を使う。多くの要件を合成する場合や構築時のクロージャアロケーションが気になる場合は `ReqExpr` を使う。

ブリッジヘルパーは両世界を整合させる。`ReifyReq` はクロージャ世界の要件を `ExprAtom` として包むため、`ReifyReq(nil)` は無効であり、評価時に panic する。`ReflectReq` は Expr 世界の要件をクロージャ経由で評価する。

## ゲート付き値

`Guard` は値と要件を対にする。`IntoView` はコンテキストが要件を満たす場合のみ `View` を返す：

```go
checked := cove.Guard(canDispatch, payload)
if view, ok := checked.IntoView(runtime);ok {
    _ = view.Extract()
}
```

`GuardRule` は名前付きルールと `Report` による診断を追加する：

```go
rule := cove.Require("budget", func(r Runtime) bool { return r.Budget > 0 })
guarded := cove.GuardRule(rule, payload)
if view, report := guarded.IntoView(runtime);report.OK() {
    _ = view.Extract()
}
```

`CheckRules` と `CheckRulesExpr` は複数のルールを評価し、最初の失敗で停止する：

```go
report := cove.CheckRules(runtime, budgetRule, permRule)
if !report.OK() {
    // report.Failed は成立しなかった最初のルール名を示す
}
```

Expr 世界の診断も `CheckRulesExpr` と `GuardRuleExpr` で同じ形になる：

```go
type Runtime struct{ Budget int }

runtime := Runtime{Budget: 1}
payload := "packet"
exprRule := cove.RequireExpr("budget", cove.ExprAtom(func(r Runtime) bool {
    return r.Budget > 0
}))
report := cove.CheckRulesExpr(runtime, exprRule)
if !report.OK() {
    return
}
guardedExpr := cove.GuardRuleExpr(exprRule, payload)
if view, report := guardedExpr.IntoView(runtime);report.OK() {
    _ = view.Extract()
}
```

Expr 世界の同等物：`GuardExpr`、`GuardRuleExpr`、`CheckedExpr`、`GuardedExpr`。Map および Pullback ヘルパー：`MapChecked`、`MapGuarded`、`PullbackChecked`、`PullbackGuarded`（およびその Expr バリアント）。

## View 操作

`View[C, A]` は Go の型の圏上の余モナドである。キャリアは直積 C × A、`Extract`（ε）は値を射影し、`Duplicate`（δ）は観察を焦点値として埋め込む：

```go
v := cove.Observe(ctx, value)
v.Ask()       // 環境コンテキスト (π₁)
v.Extract()   // 焦点値 (ε = π₂)
```

変換：`Map`（A 上の関手持ち上げ）、`MapContext`（C 上の関手持ち上げ）、`Replace`（値の置換）、`WithContext`（コンテキストの置換）。

`Duplicate` と `Extend` は余モナド律を満たす：

```go
// 余単位律：ε ∘ δ = id
cove.Duplicate(v).Extract() == v

// 余 Kleisli 恒等律：extend(ε) = id
cove.Extend(v, func(w cove.View[C, A]) A { return w.Extract() }) == v

// 結合律：extend(f) ∘ extend(g) = extend(f ∘ extend(g))
```

`Extend` は余 Kleisli 拡張である：f: W(C,A) → B が与えられたとき、環境コンテキストを保存しつつ f を W(C,A) → W(C,B) に持ち上げる。g: W(C,B) → D との間に誘導される余 Kleisli 合成は g ∘ Extend(f) である。

## エコシステムにおける位置

| パッケージ | 担当領域 | 圏論的側面 |
|------------|----------|------------|
| `kont` | エフェクト操作、ハンドラ、サスペンションと再開 | 代数 / 自由モナド / 畳み込み |
| **`cove`** | **サスペンション境界をまたぐ環境コンテキスト** | **余代数 / 余モナド / 展開** |
| `takt` | プロアクターディスパッチ、アウトカム分類、イベントループ | 余モデル（ハンドラの双対）|
| `uring` | カーネル I/O：SQE/CQE、リング管理、バッファリング | — |
| `iox` | アウトカム代数とエラーセマンティクス | — |

## 形式的構造

`View[C, A]` は環境余モナド (Uustalu & Vene 2008) であり、キャリアは C × A、余単位 ε = π₂（Extract）、余乗法 δ(c, a) = (c, (c, a)) （Duplicate）。`Extend` は余 Kleisli 拡張を実装する。

`Req[C]` と `ReqExpr[C]` は C 上の述語のブール代数の対象である。`Pullback` は射 f: C → D により誘導される反変関手 f* を実装し、D 上の述語を C 上の述語に写す。`ExprPullback` はブール代数構造を保存する：f*(p ∧ q) = f*(p) ∧ f*(q)、f*(¬p) = ¬f*(p)。

kont は代数的エフェクトをモデル化する（自由モナド、ハンドラ、畳み込み）。cove は余効果をモデル化する（余モナド、要件、展開）。両者はサスペンション境界における圏論的双対である：kont は計算が世界に何をするかを問い、cove は計算が世界から何を必要とするかを述べる。

`ReqExpr` はブール代数をタグ付きユニオン — ブール署名関手の始代数 — に脱関数化することで、要件の構造を不透明なクロージャではなく検査可能なデータにする (Reynolds 1972)。

## References

- Uustalu, T. & Vene, V. "Comonadic Notions of Computation." *CMCS 2008*, pp. 263–284. https://doi.org/10.1016/j.entcs.2008.05.029
- Petricek, T., Orchard, D. & Mycroft, A. "Coeffects: A Calculus of Context-Dependent Computation." *ICFP 2014*, pp. 123–135. https://doi.org/10.1145/2628136.2628160
- Reynolds, J.C. "Definitional Interpreters for Higher-Order Programming Languages." *ACM '72*, pp. 717–740. https://doi.org/10.1145/800194.805852

## プラットフォームサポート

`cove` はこのモジュール内の純粋 Go パッケージである。プラットフォーム固有ファイルや build tag はなく、必要条件は Go 1.26+ のみ。

## ライセンス

MIT ライセンス。詳細は [LICENSE](LICENSE) を参照。

©2026 [Hayabusa Cloud Co., Ltd.](https://code.hybscloud.com)
