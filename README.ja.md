[![Go Reference](https://pkg.go.dev/badge/code.hybscloud.com/cove.svg)](https://pkg.go.dev/code.hybscloud.com/cove)
[![Go Report Card](https://goreportcard.com/badge/github.com/hayabusa-cloud/cove)](https://goreportcard.com/report/github.com/hayabusa-cloud/cove)
[![Coverage Status](https://codecov.io/gh/hayabusa-cloud/cove/graph/badge.svg)](https://codecov.io/gh/hayabusa-cloud/cove)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

[English](README.md) | [简体中文](README.zh-CN.md) | [Español](README.es.md) | **日本語** | [Français](README.fr.md)

# cove

Go において [kont](https://code.hybscloud.com/kont) のサスペンション境界をまたいで明示的な環境コンテキストを運ぶためのコンテキスト層。

## 概要

外部ランタイムが、たとえばプロアクター、イベントループ、ディスパッチャのような形で kont
サスペンションを一つずつ進めるとき、中断された各操作には、操作自体には含まれない状態が必要になることがある。ディスパッチ予算、リングケイパビリティ、プロトコルフェーズ、バッファグループの有効性などである。明示的なコンテキストキャリアがなければ、こうした状態はその場しのぎの副マップや暗黙のグローバル変数に置かれ、合成を壊してしまう。

cove は環境コンテキストを値およびサスペンションと対にすることで、コンテキストが型付き・合成可能なデータとして非同期ステッピング境界を越えて運ばれるようにする。

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

本パッケージはポリシーフリーである。コンテキストの保持と検査のみを行い、スケジューリング・リトライ・ディスパッチは決して行わない。

## インストール

```bash
go get code.hybscloud.com/cove
```

Go 1.26+ が必要。

## コア型

| 型                                     | 用途                               |
|---------------------------------------|----------------------------------|
| `View[C, A]`                          | 余モナドキャリア：環境コンテキスト C の下の値 A       |
| `SuspensionView[C, A]`                | 環境コンテキストと対になった kont サスペンション      |
| `Req[C]`                              | C 上の反変述語（クロージャ形式）：`func(C) bool` |
| `ReqExpr[C]`                          | C 上の反変述語（脱関数化形式）                 |
| `Rule[C]` / `RuleExpr[C]`             | 診断用 `Report` 付きの名前付き述語           |
| `Checked[C, A]` / `CheckedExpr[C, A]` | 要件でゲートされた値                       |
| `Guarded[C, A]` / `GuardedExpr[C, A]` | 名前付きルールでゲートされた値                  |

`Ambient` はコンテキスト型パラメータ (C, D) を制約する。`Focus` は値型パラメータ (A, B) を制約する。

## 文脈付きステッピング

`StepWith` と `StepExprWith` は kont 計算を評価し、その最初のサスペンションを環境コンテキストと対にする。各サスペンションはアフィンであり、
`Resume`、`ResumeWith`、または `Discard` のいずれかによって、ちょうど一度だけ消費される。

```go
val, sv := cove.StepExprWith(ctx, expr)
for sv.Suspension != nil {
    op := sv.Op()
    result := handle(op)
    val, sv = sv.Resume(result)
}
```

`ResumeWith` は連続するステップの間で、運ばれてきたコンテキストを発展させる。たとえば、予算を減算し、ケイパビリティを更新し、プロトコルフェーズを進められる：

```go
val, sv = sv.ResumeWith(result, func(r Runtime) Runtime {
    r.Budget--
    return r
})
```

`MapContextSuspension` と `WithContextSuspension` は、現在のサスペンション前線を変えずに、すでに観測済みのサスペンションに載ったコンテキストを明示的に輸送または置換する：

```go
sv = cove.MapContextSuspension(sv, func(r Runtime) Runtime {
    r.Budget += 4
    return r
})
sv = cove.WithContextSuspension(sv, Runtime{Budget: 16})
```

計算が完了すると `sv.Suspension` は nil になるが、`sv.Ask()` は運ばれてきたコンテキストをなお返す。

cove は kont のステッピング分類器をそのまま転送するため、`Step`、`StepExpr`、`StepWith`、`StepExprWith`、`Resume`、
`ResumeWith` は kont の nil 完了規約を継承する。すなわち、完了値が nil の場合は `A` のゼロ値で完了したことを意味する。
したがって結果型がポインタやインタフェースである計算では、nil を意味のある完了値として用いることはできない。その区別が
必要な場合は、nil を明示的な直和型や証拠型でラップすること。担い手のコンテキストには影響しない。

`ObserveSuspension` は既存の `kont.Suspension` に、要件チェックを行わずに環境コンテキストを付与する。`CheckSuspension` と
`CheckSuspensionExpr` はそのゲート付き版である：

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

## コマンド

`Cmd[C, A, B]` は文脈コマンド `View[C, A] -> B` である。`Run` は具体的な `View` にコマンドを適用し、`ExtractCmd` は恒等コマンド、
`LiftCmd` は焦点値のみに作用する変換を文脈世界へ持ち上げ、`Compose` は `Extend` を通じてコマンドを合成する。

```go
cmd := cove.Compose(
    func(v cove.View[Runtime, int]) string {
        return fmt.Sprintf("budget=%d value=%d", v.Ask().Budget, v.Extract())
    },
    cove.LiftCmd(func(n int) int { return n + 1 }),
)
out := cove.Run(cove.Observe(Runtime{Budget: 8}, 41), cmd)
_ = out // "budget=8 value=42"
```

## 要件

要件は環境コンテキスト上の述語であり、クロージャ形式とデータ形式の両方が用意されている。どちらの形式も `All`、`Any`、`Not`
を備え、`True` と `False` はそれぞれ論理積と論理和の単位元となる。

**クロージャ形式** (`Req`)：直接的で簡潔。

```go
req := cove.All(
    cove.Req[Runtime](func(r Runtime) bool { return r.Budget > 0 }),
    cove.Req[Runtime](func(r Runtime) bool { return r.CanDispatch }),
)
ok := cove.Need(runtime, req)
```

**データ形式** (`ReqExpr`)：合成可能なブール構造で、合成時のクロージャアロケーションを回避する。

```go
expr := cove.ExprAll(
    cove.ExprAtom(func(r Runtime) bool { return r.Budget > 0 }),
    cove.ExprAtom(func(r Runtime) bool { return r.CanDispatch }),
)
ok := cove.NeedExpr(runtime, expr)
```

コンビネータ：`All`/`ExprAll`（論理積 ∧）、`Any`/`ExprAny`（論理和 ∨）、`Not`/`ExprNot`（否定 ¬）、`True`/`ExprTrue`（⊤）、`False`/`ExprFalse`（⊥）。

`Pullback` と `ExprPullback` はコンテキスト射に沿って要件を反変に輸送する。`f: C → D` が与えられたとき、`Pullback(req, f)`
は `Req[D]` を `Req[C]` に写す。`ExprPullback` はブール構造を保存し、`ExprNot`、`ExprAll`、`ExprAny` に対して分配する。

### Req と ReqExpr の使い分け

その場限りで一回きりの述語には `Req` を使う。多くの要件を合成する場合や、構築時のクロージャアロケーションが気になる場合には
`ReqExpr` を使う。

ブリッジヘルパーは両形式を整合させる：`ReifyReq` はクロージャ形式の要件をひとつの `ExprAtom` として包むため、
`ReifyReq(nil)` は無効であり、評価時に panic する。`ReflectReq` は Expr 形式の要件をクロージャ経由で評価する。

## ゲート付き値

`Guard` は値とひとつの要件を対にする。`IntoView` は、環境コンテキストがその要件を満たす場合に限り `View` を返す：

```go
checked := cove.Guard(canDispatch, payload)
if view, ok := checked.IntoView(runtime); ok {
    _ = view.Extract()
}
```

`GuardRule` は名前付きルールと `Report` による診断を追加する：

```go
rule := cove.Require("budget", func(r Runtime) bool { return r.Budget > 0 })
guarded := cove.GuardRule(rule, payload)
if view, report := guarded.IntoView(runtime); report.OK() {
    _ = view.Extract()
}
```

`CheckRules` と `CheckRulesExpr` は複数のルールを順に評価し、最初の失敗で停止する：

```go
report := cove.CheckRules(runtime, budgetRule, permRule)
if !report.OK() {
    // report.Failed は成立しなかった最初のルール名を示す
}
```

Expr 形式の診断経路も、`CheckRulesExpr` と `GuardRuleExpr` によって同じ形で提供される：

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
if view, report := guardedExpr.IntoView(runtime); report.OK() {
    _ = view.Extract()
}
```

Expr 形式の対応物：`GuardExpr`、`GuardRuleExpr`、`CheckedExpr`、`GuardedExpr`。Map および Pullback ヘルパー：`MapChecked`、
`MapGuarded`、`PullbackChecked`、`PullbackGuarded`（およびその Expr バリアント）。

## View 操作

`View[C, A]` は Go の型の圏上の余モナドである。キャリアは直積 C × A、`Extract`（ε）は値を射影し、`Duplicate`（δ）は観察を焦点値として埋め込む：

```go
v := cove.Observe(ctx, value)
v.Ask()       // 環境コンテキスト (π₁)
v.Extract()   // 焦点値 (ε = π₂)
```

変換：`Map`（`A` 上の関手持ち上げ）、`MapContext`（`C` 上の関手持ち上げ）、`Replace`（値の置換）、`WithContext`（コンテキストの置換）。

`Duplicate` と `Extend` は余モナド律を満たす：

```go
// 余単位律：ε ∘ δ = id
cove.Duplicate(v).Extract() == v

// 拡張恒等律：extend(ε) = id
cove.Extend(v, func(w cove.View[C, A]) A { return w.Extract() }) == v

// 結合律：extend(f) ∘ extend(g) = extend(f ∘ extend(g))
```

`Extend` は文脈拡張演算子である：`f: W(C, A) → B` が与えられたとき、環境コンテキストを保存しつつ `f` を
`W(C, A) → W(C, B)` に持ち上げる。`g: W(C, B) → D` との間に誘導されるコマンド合成は `g ∘ Extend(f)` である。

## エコシステムにおける位置

| パッケージ | 担当領域 | 圏論的側面 |
|------------|----------|------------|
| `kont` | エフェクト操作、ハンドラ、サスペンションと再開 | 代数 / 自由モナド / 畳み込み |
| **`cove`** | **サスペンション境界をまたぐ環境コンテキスト** | **余代数 / 余モナド / 展開** |
| `takt` | プロアクターディスパッチ、アウトカム分類、イベントループ | 余モデル（ハンドラの双対）|
| `uring` | カーネル I/O：SQE/CQE、リング管理、バッファリング | — |
| `iox` | アウトカム代数とエラーセマンティクス | — |

## 形式的構造

`View[C, A]` は環境余モナド (Uustalu & Vene 2008) であり、キャリアは C × A、余単位 ε = π₂（Extract）、余乗法 δ(c, a) = (c, (
c, a)) （Duplicate）。`Extend` は文脈拡張（圏論では coKleisli 拡張）を実装する。

`Req[C]` と `ReqExpr[C]` は C 上の述語のブール代数の対象である。`Pullback` は射 f: C → D により誘導される反変関手 f* を実装し、D 上の述語を C 上の述語に写す。`ExprPullback` はブール代数構造を保存する：f*(p ∧ q) = f*(p) ∧ f*(q)、f*(¬p) = ¬f*(p)。

kont は代数的エフェクトをモデル化する（自由モナド、ハンドラ、畳み込み）。cove
は余効果をモデル化する（余モナド、要件、展開）。両者はサスペンション境界における圏論的双対である：kont は計算が世界に対して何をするかを述べ、cove
は計算が世界から何を必要とするかを述べる。

モーダルエフェクトの視座から読むと、キャリア形状は周囲のコンテキストに何が起こるかを名指すものとして読める：kont（C →
D）はそれを置き換え、cove（W(C, A) → B）はそれを相対的に扱い、純粋な素通しはそれを保存する。エフェクト構造は関数の「色」ではなくキャリアに乗るため、cove
はポリシーに依らない状態を保てる。

`ReqExpr`
はブール代数を——ブール署名関手の始代数である——タグ付きユニオンへ脱関数化することで、要件の構造を不透明なクロージャではなく検査可能なデータにする (
Reynolds 1972)。

## 実用レシピ

完全な文脈依存計算は通常、三つの段階を経る：要件を宣言し、その要件で値をゲートし、最後に具体的な文脈の下で結果を観測する。

```go
// 1. 環境文脈に対する要件を宣言する。
type Caps struct{ CanSubmit, HasToken bool }
req := cove.All(
    func(c Caps) bool { return c.CanSubmit },
    func(c Caps) bool { return c.HasToken },
)

// 2. その要件で値をゲートする。これで Checked[Caps, T] が生成される。
checked := cove.Guard(req, payload)

// 3. Expr に具体化し、異なる環境文脈の下でステップ実行できるようにする。
expr := cove.ReifyReq(req)
ok   := cove.NeedExpr(Caps{CanSubmit: true, HasToken: true}, expr)
_    = checked
_    = ok
```

`Pullback` を用いると、ある文脈上で定義された要件を別の文脈上で動作するように適応できる（`Pullback(req, f)`、`f: D → C`）。
`MapChecked` と `MapGuarded` は述語を再検査することなく、値レベルの変換を通してゲート済みの値を運ぶ。`All` + `Pullback` +
`Guard` の組み合わせは、下流パッケージが特定の文脈型と結合することなく、型付きの能力チェックを構築するための手段である。

## 参考文献

- John C. Reynolds. 1972. Definitional Interpreters for Higher-Order Programming Languages. In *Proc. ACM Annual
  Conference (ACM '72)*. 717–740. https://doi.org/10.1145/800194.805852
- Tarmo Uustalu and Varmo Vene. 2008. Comonadic Notions of Computation. *Electronic Notes in Theoretical Computer
  Science* 203, 5 (June 2008), 263–284. https://doi.org/10.1016/j.entcs.2008.05.029
- Tomas Petricek, Dominic Orchard, and Alan Mycroft. 2014. Coeffects: A Calculus of Context-Dependent Computation. In
  *Proc. 19th ACM SIGPLAN International Conference on Functional Programming (ICFP '14)*.
  123–135. https://tomasp.net/academic/papers/structural/coeffects-icfp.pdf
- Marco Gaboardi, Shin-ya Katsumata, Dominic Orchard, Flavien Breuvart, and Tarmo Uustalu. 2016. Combining Effects and
  Coeffects via Grading. In *Proc. 21st ACM SIGPLAN International Conference on Functional Programming (ICFP '16)*.
  476–489. https://doi.org/10.1145/2951913.2951939
- Jonathan Immanuel Brachthäuser, Philipp Schuster, and Klaus Ostermann. 2020. Effects as Capabilities: Effect Handlers
  and Lightweight Effect Polymorphism. *Proc. ACM on Programming Languages* 4, OOPSLA (Nov. 2020), Article 126, 30
  pages. https://se.cs.uni-tuebingen.de/publications/brachthaeuser20effekt.pdf
- Daniel Gratzer, G. A. Kavvos, Andreas Nuyts, and Lars Birkedal. 2021. Multimodal Dependent Type Theory. *Logical
  Methods in Computer Science* 17, 3 (2021), Paper 11, 67 pages. https://doi.org/10.46298/lmcs-17(3:11)2021
- Wenhao Tang, Leo White, Stephen Dolan, Daniel Hillerström, Sam Lindley, and Anton Lorenzen. 2025. Modal Effect Types.
  *Proc. ACM Program. Lang.* 9, OOPSLA1 (Apr. 2025), Article 120, 1130–1157. https://arxiv.org/abs/2407.11816
- Wenhao Tang and Sam Lindley. 2026. Rows and Capabilities as Modal Effects. *Proc. ACM Program. Lang.* 10, POPL (Jan.
  2026), 923–950. https://arxiv.org/abs/2507.10301

## プラットフォームサポート

`cove` はこのモジュール内の純粋 Go パッケージである。プラットフォーム固有ファイルや build tag はなく、必要条件は Go 1.26+ のみ。

## ライセンス

MIT ライセンス。詳細は [LICENSE](LICENSE) を参照。

©2026 [Hayabusa Cloud Co., Ltd.](https://code.hybscloud.com)
