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

| 型                                                   | 用途                                |
|-----------------------------------------------------|-----------------------------------|
| `View[C, A]`                                        | 余モナドキャリア：環境コンテキスト C の下の値 A        |
| `SuspensionView[C, A]`                              | 環境コンテキストと対になった kont サスペンション       |
| `World`                                             | 環境コンテキスト型の Kripke 的読み方            |
| `StepIndex`                                         | ステップ添字観測のための有限近似レベル               |
| `Preorder[C]`                                       | Kripke 世界拡張関係 `w <= w'` と局所的な法則検査 |
| `Transition[C]`                                     | 2 つの Kripke 世界の具体的な候補エッジ          |
| `Forcing[C]` / `ForcingExpr[C]`                     | 世界前順序と単調性チェックを持つ要件                |
| `Relation[C, A]`                                    | 世界・添字・値に対するステップ添字 Kripke 関係       |
| `IndexedView[C, A]` / `IndexedSuspensionView[C, A]` | 明示的なステップ credit を持つ文脈観測           |
| `Req[C]`                                            | C 上の反変述語（クロージャ形式）：`func(C) bool`  |
| `ReqExpr[C]`                                        | C 上の反変述語（脱関数化形式）                  |
| `Rule[C]` / `RuleExpr[C]`                           | 診断用 `Report` 付きの名前付き述語            |
| `Checked[C, A]` / `CheckedExpr[C, A]`               | 要件でゲートされた値                        |
| `Guarded[C, A]` / `GuardedExpr[C, A]`               | 名前付きルールでゲートされた値                   |

`Ambient` はコンテキスト型パラメータ (C, D) を制約する。`World` は同じ構造制約に Kripke 語彙を与え、`Focus`
は値型パラメータ (A, B) を制約する。

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

cove は kont のステッピング分類を使用する。nil サスペンションが完了を示し、返される値が完了結果となる。nil を許容する型では nil も有効な結果であり、型付き nil を格納したインタフェースはその動的型を保持する。中断中のステップも `A` のゼロ値を返すことがあるため、完了判定にはサスペンションを使う。保持しているコンテキストは変わらない。

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

チェックが拒否されても元のサスペンションは消費されない。再チェックまたは明示的な `Discard` まで保持する。
所有者からコンテキストを更新し、コールバックの再入も含め、許可した操作まで条件を保護する。
`ResumeWith` のマッパーは完了時も含めて継続を進めた後に実行される。`ResumeTo` は渡された世界エッジを検査するだけで、
新しい観測の取得や任意の要件の検査は行わない。取り消せる適格性は永続的な forcing ではなく、`StepIndex` は時間ではなく論理的なクレジットである。
ディスパッチ、継続の進行、外部リソースの解放には別々の契約があり、`Discard` は I/O を完了させない。
[時間的な観測の例](temporal_example_test.go) は外部リソースを持たない純粋な操作で、拒否、更新、完了を示す。

## ステップ添字 Kripke 証拠

`cove` はスケジューラ方針を持ち込まずに Kripke 構造を明示できる。`Preorder[C]` は世界拡張を定義し、`Extends`、
`ReflexiveAt`、`TransitiveAt` の局所検査を提供する。`Transition[C]`、`CheckTransition`、`CheckForcingTransition`、
`CheckInvariantTransition` は具体的な世界エッジを検査する。`Forcing` は要件をその前順序と対にし、`Relation` は世界・有限
`StepIndex`・値によって意味的妥当性を添字付ける。

`DiscreteWorlds` は等しい世界だけを関連付ける場合に使い、`TotalWorlds` はすべての世界対を関連付ける前順序を与える。
`ForceExpr` は `ReqExpr` 用の `Force` であり、`CheckRelation`、`WeakenIndexedView`、`WeakenIndexedSuspension` は関係検査とステップ
credit の弱化を明示する。

```go
type RuntimeWorld struct {
    Epoch uint64
}

leq := func(w, next RuntimeWorld) bool {
    return w.Epoch <= next.Epoch
}
_ = cove.Preorder[RuntimeWorld](leq).ReflexiveAt(RuntimeWorld{Epoch: 1})
_ = cove.Preorder[RuntimeWorld](leq).TransitiveAt(RuntimeWorld{Epoch: 1}, RuntimeWorld{Epoch: 2}, RuntimeWorld{Epoch: 3})

canObserve := cove.Force(leq, func(w RuntimeWorld) bool {
    return w.Epoch > 0
})
_ = canObserve.MonotoneAt(RuntimeWorld{Epoch: 1}, RuntimeWorld{Epoch: 2})

rel := cove.Relate(leq, func(w RuntimeWorld, n cove.StepIndex, value int) bool {
    return uint64(n) <= w.Epoch && value >= 0
})
later := cove.Later(rel)
_ = later.Holds(RuntimeWorld{Epoch: 8}, 3, 4)
```

操作的な prefix では、`StepWithIndex` と `StepExprWithIndex` が `SuspensionView` を明示的な fuel と対にする。indexed
`Resume` ごとに 1 ステップ分の credit を消費するため、有限観測が可視的に減少する。後続コンテキストを Kripke 世界拡張として検査する場合は
`ResumeTo` を使い、完了済み indexed frontier の最終関係は `CheckCompletedRelation` で検査する：

```go
_, sv := cove.StepExprWithIndex(RuntimeWorld{Epoch: 2}, 2, expr)
var val int
for sv.Extract() != nil {
    result := dispatch(sv.Ask(), sv.Op())
    next := sv.Ask()
    next.Epoch++
    val, sv = sv.ResumeTo(leq, result, next)
}
_ = cove.CheckCompletedRelation(sv, val, rel)
```

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

固定した C と純粋なコールバックの下で、`View[C, A]` は後述する環境余モナドの構造を持つ。キャリアは直積 C × A、`Extract`（ε）は値を射影し、`Duplicate`（δ）は観察を焦点値として埋め込む：

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

| パッケージ      | 担当領域                           | 圏論的側面               |
|------------|--------------------------------|---------------------|
| `kont`     | エフェクト操作、ハンドラ、サスペンションと再開        | 代数 / 自由モナド / 畳み込み   |
| **`cove`** | **サスペンション境界をまたぐ環境コンテキスト**      | **余代数 / 余モナド / 展開** |
| `takt`     | プロアクターディスパッチ、ステップ実行とランナー観測     | 余モデル（ハンドラの双対）       |
| `uring`    | カーネル I/O：SQE/CQE、リング管理、バッファリング | —                   |
| `iox`      | アウトカム代数とエラーセマンティクス             | —                   |

## 形式的構造

固定した C と純粋な全域コールバックの下で、`View[C, A]` は環境余モナド (Uustalu & Vene 2008) となる。キャリアは C × A、余単位は ε = π₂（`Extract`）、余乗法は δ(c, a) = (c, (c, a))（`Duplicate`）。`Extend` は coKleisli 拡張を実装する。等式は安定したコンテキストにおける観測的等価性を表し、値や関数が Go で比較可能であることは要求しない。

`Req[C]` は述語を表し、`ReqExpr[C]` は不透明な関数を原子とするブール構文を保持する。述語と射影が純粋な場合、`Pullback` は f: C → D による反変写像 f* を実装する。`ExprPullback` はブール構造を保存する：f*(p ∧ q) = f*(p) ∧ f*(q)、f*(¬p) = ¬f*(p)。評価中に訪れた各原子に対して射影を呼び出す場合がある。

kont はエフェクトを伴う計算と再開操作を提供し、cove はその境界を観測・検査するためのコンテキストを保持する。事実の取得、同期、実行およびリソースの寿命管理は呼び出し側が担当する。

複合要件は左から右へ短絡評価される。`All`、`Any`、`ExprAll`、`ExprAny` はスライスで渡された複数のオペランドを保持するため、使用中はそのストレージを変更しない。View のコピーは参照先のストレージを共有する。`ExprAtom` と `ReifyReq` には非 nil の述語を渡す。

`ReqExpr` はブール合成をデータとして表現し、葉の述語は Go 関数のまま保持する。評価はこの構造を解釈して実行される。`ReifyReq` は不透明な述語を原子として包む操作であり、クロージャから構文を抽出したり、実行を自動的に特殊化したりはしない。

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

// 3. 具体的なコンテキストで要件を確認してから値を使う。
if view, ok := checked.IntoView(Caps{CanSubmit: true, HasToken: true}); ok {
	_ = view.Extract()
}
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
