[![Go Reference](https://pkg.go.dev/badge/code.hybscloud.com/cove.svg)](https://pkg.go.dev/code.hybscloud.com/cove)
[![Go Report Card](https://goreportcard.com/badge/github.com/hayabusa-cloud/cove)](https://goreportcard.com/report/github.com/hayabusa-cloud/cove)
[![Coverage Status](https://codecov.io/gh/hayabusa-cloud/cove/graph/badge.svg)](https://codecov.io/gh/hayabusa-cloud/cove)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

[English](README.md) | **简体中文** | [Español](README.es.md) | [日本語](README.ja.md) | [Français](README.fr.md)

# cove

用于在 Go 中跨越 [kont](https://code.hybscloud.com/kont) 挂起边界传递显式环境上下文的上下文层。

## 概览

当外部运行时（例如主动器、事件循环或分派器）逐个推进 kont
挂起时，每个被挂起的操作都可能需要操作本身不包含的状态：分派预算、环能力、协议阶段、缓冲组有效性。没有显式的上下文载体，这些状态便会落入临时的侧映射或隐式全局变量，破坏可组合性。

cove 将环境上下文与值及挂起配对，使上下文以类型化、可组合的数据形式跨越异步步进边界。

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

本包是无策略的：只承载与检查上下文，从不调度、重试或分派。

## 安装

```bash
go get code.hybscloud.com/cove
```

需要 Go 1.26+。

## 核心类型

| 类型                                    | 用途                            |
|---------------------------------------|-------------------------------|
| `View[C, A]`                          | 余单子载体：环境上下文 C 下的值 A           |
| `SuspensionView[C, A]`                | 与环境上下文配对的 kont 挂起             |
| `Req[C]`                              | C 上的逆变谓词（闭包形式）：`func(C) bool` |
| `ReqExpr[C]`                          | C 上的逆变谓词（去函数化形式）              |
| `Rule[C]` / `RuleExpr[C]`             | 带诊断 `Report` 的命名谓词            |
| `Checked[C, A]` / `CheckedExpr[C, A]` | 由需求门控的值                       |
| `Guarded[C, A]` / `GuardedExpr[C, A]` | 由命名规则门控的值                     |

`Ambient` 约束上下文类型参数 (C, D)。`Focus` 约束值类型参数 (A, B)。

## 上下文步进

`StepWith` 与 `StepExprWith` 对 kont 计算求值，并将其第一个挂起与环境上下文配对。每个挂起都是仿射的：须通过 `Resume`、
`ResumeWith` 或 `Discard` 恰好消耗一次。

```go
val, sv := cove.StepExprWith(ctx, expr)
for sv.Suspension != nil {
    op := sv.Op()
    result := handle(op)
    val, sv = sv.Resume(result)
}
```

`ResumeWith` 在相邻步之间演化所携带的上下文。例如，它可以递减预算、更新能力或推进协议阶段：

```go
val, sv = sv.ResumeWith(result, func(r Runtime) Runtime {
    r.Budget--
    return r
})
```

`MapContextSuspension` 与 `WithContextSuspension` 在不改变当前挂起前沿的前提下，显式搬运或替换已观察挂起所携带的上下文：

```go
sv = cove.MapContextSuspension(sv, func(r Runtime) Runtime {
    r.Budget += 4
    return r
})
sv = cove.WithContextSuspension(sv, Runtime{Budget: 16})
```

计算完成后，`sv.Suspension` 变为 nil，但 `sv.Ask()` 仍返回所携带的上下文。

`ObserveSuspension` 在不进行任何需求检查的前提下，为已有的 `kont.Suspension` 附加环境上下文。`CheckSuspension` 与
`CheckSuspensionExpr` 提供带门控的对应版本：

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

`Step` 和 `StepExpr` 重新导出 kont 的评估器，不绑定上下文。

## 命令

`Cmd[C, A, B]` 是上下文命令 `View[C, A] -> B`。`Run` 将命令应用到具体的 `View`；`ExtractCmd` 是恒等命令；`LiftCmd`
把仅作用于焦点值的映射提升到上下文世界；`Compose` 通过 `Extend` 组合命令。

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

## 需求

需求是环境上下文上的谓词，提供闭包形式与数据形式。两种形式都支持 `All`、`Any`、`Not`，其中 `True` 与 `False` 分别是合取与析取的幺元。

**闭包形式** (`Req`)：直接简洁。

```go
req := cove.All(
    cove.Req[Runtime](func(r Runtime) bool { return r.Budget > 0 }),
    cove.Req[Runtime](func(r Runtime) bool { return r.CanDispatch }),
)
ok := cove.Need(runtime, req)
```

**数据形式** (`ReqExpr`)：可组合的布尔结构，在组合时避免闭包分配。

```go
expr := cove.ExprAll(
    cove.ExprAtom(func(r Runtime) bool { return r.Budget > 0 }),
    cove.ExprAtom(func(r Runtime) bool { return r.CanDispatch }),
)
ok := cove.NeedExpr(runtime, expr)
```

组合子：`All`/`ExprAll`（合取 ∧）、`Any`/`ExprAny`（析取 ∨）、`Not`/`ExprNot`（否定 ¬）、`True`/`ExprTrue`（⊤）、`False`/`ExprFalse`（⊥）。

`Pullback` 与 `ExprPullback` 沿上下文投影以逆变方式传输需求。给定 `f: C → D`，`Pullback(req, f)` 将 `Req[D]` 映射为
`Req[C]`。`ExprPullback` 保持布尔结构，并对 `ExprNot`、`ExprAll`、`ExprAny` 进行分配。

### 选择 Req 还是 ReqExpr

临时、一次性的谓词使用 `Req`；当需要组合许多需求、或关注构造阶段的闭包分配时，使用 `ReqExpr`。

桥接辅助函数使两种形式保持对齐：`ReifyReq` 将闭包形式的需求包装为一个 `ExprAtom`，因此 `ReifyReq(nil)` 无效，求值时会
panic；`ReflectReq` 通过闭包对 Expr 形式的需求进行求值。

## 门控值

`Guard` 将值与一个需求配对；`IntoView` 仅当环境上下文满足该需求时才返回 `View`：

```go
checked := cove.Guard(canDispatch, payload)
if view, ok := checked.IntoView(runtime); ok {
    _ = view.Extract()
}
```

`GuardRule` 通过命名规则和 `Report` 添加诊断：

```go
rule := cove.Require("budget", func(r Runtime) bool { return r.Budget > 0 })
guarded := cove.GuardRule(rule, payload)
if view, report := guarded.IntoView(runtime); report.OK() {
    _ = view.Extract()
}
```

`CheckRules` 与 `CheckRulesExpr` 依次对多条规则求值，在首个失败处停止：

```go
report := cove.CheckRules(runtime, budgetRule, permRule)
if !report.OK() {
    // report.Failed 是第一个不成立规则的名称
}
```

Expr 形式的诊断路径形状相同，由 `CheckRulesExpr` 与 `GuardRuleExpr` 提供：

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

Expr 形式的对应物：`GuardExpr`、`GuardRuleExpr`、`CheckedExpr`、`GuardedExpr`。Map 与 pullback 辅助函数：`MapChecked`、
`MapGuarded`、`PullbackChecked`、`PullbackGuarded`（以及其 Expr 变体）。

## View 操作

`View[C, A]` 是 Go 类型范畴上的余单子。载体为积 C × A，`Extract`（ε）投影值，`Duplicate`（δ）将观察嵌入为焦点值：

```go
v := cove.Observe(ctx, value)
v.Ask()       // 环境上下文 (π₁)
v.Extract()   // 聚焦值 (ε = π₂)
```

变换：`Map`（在 `A` 上的函子提升）、`MapContext`（在 `C` 上的函子提升）、`Replace`（替换值）、`WithContext`（替换上下文）。

`Duplicate` 和 `Extend` 满足余单子定律：

```go
// 余单位律：ε ∘ δ = id
cove.Duplicate(v).Extract() == v

// 扩展恒等律：extend(ε) = id
cove.Extend(v, func(w cove.View[C, A]) A { return w.Extract() }) == v

// 结合律：extend(f) ∘ extend(g) = extend(f ∘ extend(g))
```

`Extend` 是上下文扩展算子：给定 `f: W(C, A) → B`，在保持环境上下文的同时将 `f` 提升为 `W(C, A) → W(C, B)`。与
`g: W(C, B) → D` 诱导出的命令组合为 `g ∘ Extend(f)`。

## 生态系统定位

| 包 | 职责 | 范畴性质 |
|----|------|----------|
| `kont` | 效应操作、处理器、挂起与恢复 | 代数 / 自由单子 / 折叠 |
| **`cove`** | **跨挂起边界的环境上下文** | **余代数 / 余单子 / 展开** |
| `takt` | 主动器分派、结果分类、事件循环 | 余模型（处理器对偶）|
| `uring` | 内核 I/O：SQE/CQE、环管理、缓冲环 | — |
| `iox` | 结果代数与错误语义 | — |

## 形式结构

`View[C, A]` 是环境余单子 (Uustalu & Vene 2008)，载体为 C × A，余单位 ε = π₂（Extract），余乘法 δ(c, a) = (c, (c, a))
（Duplicate）。`Extend` 实现上下文扩展（范畴论中的余 Kleisli 扩展）。

`Req[C]` 和 `ReqExpr[C]` 是 C 上谓词布尔代数中的对象。`Pullback` 实现由态射 f: C → D 诱导的逆变函子 f*，将 D 上的谓词映射为 C 上的谓词。`ExprPullback` 保持布尔代数结构：f*(p ∧ q) = f*(p) ∧ f*(q)，f*(¬p) = ¬f*(p)。

kont 建模代数效应（自由单子、处理器、折叠）；cove 建模余效应（余单子、需求、展开）。二者在挂起边界上互为范畴对偶：kont
描述计算对世界做了什么，cove 陈述计算从世界需要什么。

从模态效应（modal effects）的角度看，载体形状可读作对环境上下文之变化的命名：kont（C → D）替换之，cove（W(C, A) →
B）以相对方式处理之，纯透传则保持之。效应结构因此搭载于载体之上，而非函数的"颜色"之上——这正是 cove 保持策略无关的原因。

`ReqExpr` 将布尔代数去函数化为标签联合——布尔签名函子的始代数——使需求的结构成为可检查的数据，而非不透明的闭包 (Reynolds
1972)。

## 实用范式

一个完整的上下文相关计算通常经由三个阶段：声明需求、依需求守护值，然后在具体上下文下加以观测。

```go
// 1. 在环境上下文上声明一个需求。
type Caps struct{ CanSubmit, HasToken bool }
req := cove.All(
    func(c Caps) bool { return c.CanSubmit },
    func(c Caps) bool { return c.HasToken },
)

// 2. 用该需求守护一个值——产生 Checked[Caps, T]。
checked := cove.Guard(req, payload)

// 3. 物化为 Expr，从而能在不同的环境上下文下逐步求值。
expr := cove.ReifyReq(req)
ok   := cove.NeedExpr(Caps{CanSubmit: true, HasToken: true}, expr)
_    = checked
_    = ok
```

`Pullback` 让调用者把定义在某个上下文上的需求适配到另一个上下文（`Pullback(req, f)`，其中 `f: D → C`）；`MapChecked` 与
`MapGuarded` 可在不重新检查谓词的前提下，沿值层面的变换搬运被守护的值。`All` + `Pullback` + `Guard`
这一组合，正是下游包用以构建带类型的能力检查、又不与任一具体上下文类型耦合的方式。

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

## 平台支持

`cove` 是该模块中的纯 Go 包。它没有平台特定文件或 build tag；模块要求仅为 Go 1.26+。

## 许可证

MIT 许可证。详见 [LICENSE](LICENSE)。

©2026 [Hayabusa Cloud Co., Ltd.](https://code.hybscloud.com)
