[![Go Reference](https://pkg.go.dev/badge/code.hybscloud.com/cove.svg)](https://pkg.go.dev/code.hybscloud.com/cove)
[![Go Report Card](https://goreportcard.com/badge/github.com/hayabusa-cloud/cove)](https://goreportcard.com/report/github.com/hayabusa-cloud/cove)
[![Coverage Status](https://codecov.io/gh/hayabusa-cloud/cove/graph/badge.svg)](https://codecov.io/gh/hayabusa-cloud/cove)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

[English](README.md) | **简体中文** | [Español](README.es.md) | [日本語](README.ja.md) | [Français](README.fr.md)

# cove

Go 中 [kont](https://code.hybscloud.com/kont) 挂起的余代数上下文层，是 kont 代数效应的余效应对偶。

## 概览

当外部运行时（如主动器、事件循环或分派器）逐个处理 kont 挂起时，每个挂起的操作可能需要操作本身不包含的状态：分派预算、环能力、协议阶段、缓冲组有效性。没有显式的上下文载体，这些状态就会被放在临时的侧映射或隐式全局变量中，破坏可组合性。

cove 将环境上下文与值和挂起配对，使上下文以类型化、可组合的数据形式通过异步步进边界。

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

包是无策略的：只承载和检查上下文，不调度、不重试、不分派。

## 安装

```bash
go get code.hybscloud.com/cove
```

需要 Go 1.26+。

## 核心类型

| 类型 | 用途 |
|------|------|
| `View[C, A]` | 余单子载体：环境上下文 C 下的值 A |
| `SuspensionView[C, A]` | 与环境上下文配对的 kont 挂起 |
| `Req[C]` | C 上的逆变谓词函子（闭包形式）：`func(C) bool` |
| `ReqExpr[C]` | C 上的逆变谓词函子（去函数化形式）|
| `Rule[C]` / `RuleExpr[C]` | 带诊断 `Report` 的命名谓词 |
| `Checked[C, A]` / `CheckedExpr[C, A]` | 由需求门控的值 |
| `Guarded[C, A]` / `GuardedExpr[C, A]` | 由命名规则门控的值 |

`Ambient` 约束上下文类型参数 (C, D)。`Focus` 约束值类型参数 (A, B)。

## 上下文步进

`StepWith` 和 `StepExprWith` 评估 kont 计算并将第一个挂起与环境上下文配对。每个挂起是仿射的：通过 `Resume`、`ResumeWith` 或 `Discard` 恰好消耗一次。

```go
val, sv := cove.StepExprWith(ctx, expr)
for sv.Suspension != nil {
    op := sv.Op()
    result := handle(op)
    val, sv = sv.Resume(result)
}
```

`ResumeWith` 在步进间更新上下文，例如递减预算、更新能力或推进协议阶段：

```go
val, sv = sv.ResumeWith(result, func(r Runtime) Runtime {
    r.Budget--
    return r
})
```

`ObserveSuspension` 会在不检查需求的情况下为已有 `kont.Suspension` 附加环境上下文。`CheckSuspension` 和 `CheckSuspensionExpr` 则提供带门控的版本：

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

## 需求

需求是上下文上的谓词，提供闭包形式和数据形式。两者都支持 `All`/`Any`/`Not`，其中 `True` 和 `False` 分别是合取与析取的幺元。

**闭包形式** (`Req`) — 直接简洁：

```go
req := cove.All(
    cove.Req[Runtime](func(r Runtime) bool { return r.Budget > 0 }),
    cove.Req[Runtime](func(r Runtime) bool { return r.CanDispatch }),
)
ok := cove.Need(runtime, req)
```

**数据形式** (`ReqExpr`) — 可组合的布尔结构，组合时避免闭包分配：

```go
expr := cove.ExprAll(
    cove.ExprAtom(func(r Runtime) bool { return r.Budget > 0 }),
    cove.ExprAtom(func(r Runtime) bool { return r.CanDispatch }),
)
ok := cove.NeedExpr(runtime, expr)
```

组合子：`All`/`ExprAll`（合取 ∧）、`Any`/`ExprAny`（析取 ∨）、`Not`/`ExprNot`（否定 ¬）、`True`/`ExprTrue`（⊤）、`False`/`ExprFalse`（⊥）。

`Pullback` 和 `ExprPullback` 沿上下文投影以逆变方式传输需求。给定 f: C → D，`Pullback(req, f)` 将 `Req[D]` 映射为 `Req[C]`。`ExprPullback` 保持布尔结构，并对 `ExprNot`、`ExprAll`、`ExprAny` 分配。

### 选择 Req 还是 ReqExpr

临时一次性谓词用 `Req`。组合多个需求或关注构造时闭包分配时用 `ReqExpr`。

桥接辅助函数让两个世界保持对齐：`ReifyReq` 将闭包世界的需求包装为 `ExprAtom`，因此 `ReifyReq(nil)` 是无效的，并会在求值时 panic；`ReflectReq` 通过闭包求值 Expr 世界的需求。

## 门控值

`Guard` 将值与需求配对；`IntoView` 仅当上下文满足需求时返回 `View`：

```go
checked := cove.Guard(canDispatch, payload)
if view, ok := checked.IntoView(runtime);ok {
    _ = view.Extract()
}
```

`GuardRule` 通过命名规则和 `Report` 添加诊断：

```go
rule := cove.Require("budget", func(r Runtime) bool { return r.Budget > 0 })
guarded := cove.GuardRule(rule, payload)
if view, report := guarded.IntoView(runtime);report.OK() {
    _ = view.Extract()
}
```

`CheckRules` 和 `CheckRulesExpr` 评估多个规则，在第一个失败处停止：

```go
report := cove.CheckRules(runtime, budgetRule, permRule)
if !report.OK() {
    // report.Failed 是第一个不成立规则的名称
}
```

Expr 世界的诊断路径与此相同，可用 `CheckRulesExpr` 和 `GuardRuleExpr`：

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

Expr 世界等价物：`GuardExpr`、`GuardRuleExpr`、`CheckedExpr`、`GuardedExpr`。Map 和 Pullback 辅助函数：`MapChecked`、`MapGuarded`、`PullbackChecked`、`PullbackGuarded`（及其 Expr 变体）。

## View 操作

`View[C, A]` 是 Go 类型范畴上的余单子。载体为积 C × A，`Extract`（ε）投影值，`Duplicate`（δ）将观察嵌入为焦点值：

```go
v := cove.Observe(ctx, value)
v.Ask()       // 环境上下文 (π₁)
v.Extract()   // 聚焦值 (ε = π₂)
```

变换：`Map`（A 上的函子提升）、`MapContext`（C 上的函子提升）、`Replace`（替换值）、`WithContext`（替换上下文）。

`Duplicate` 和 `Extend` 满足余单子定律：

```go
// 余单位律：ε ∘ δ = id
cove.Duplicate(v).Extract() == v

// 余 Kleisli 恒等律：extend(ε) = id
cove.Extend(v, func(w cove.View[C, A]) A { return w.Extract() }) == v

// 结合律：extend(f) ∘ extend(g) = extend(f ∘ extend(g))
```

`Extend` 是余 Kleisli 扩展：给定 f: W(C,A) → B，在保持环境上下文的同时将 f 提升为 W(C,A) → W(C,B)。与 g: W(C,B) → D 诱导出的余 Kleisli 组合是 g ∘ Extend(f)。

## 生态系统定位

| 包 | 职责 | 范畴性质 |
|----|------|----------|
| `kont` | 效应操作、处理器、挂起与恢复 | 代数 / 自由单子 / 折叠 |
| **`cove`** | **跨挂起边界的环境上下文** | **余代数 / 余单子 / 展开** |
| `takt` | 主动器分派、结果分类、事件循环 | 余模型（处理器对偶）|
| `uring` | 内核 I/O：SQE/CQE、环管理、缓冲环 | — |
| `iox` | 结果代数与错误语义 | — |

## 形式结构

`View[C, A]` 是环境余单子 (Uustalu & Vene 2008)，载体为 C × A，余单位 ε = π₂（Extract），余乘法 δ(c, a) = (c, (c, a)) （Duplicate）。`Extend` 实现余 Kleisli 扩展。

`Req[C]` 和 `ReqExpr[C]` 是 C 上谓词布尔代数中的对象。`Pullback` 实现由态射 f: C → D 诱导的逆变函子 f*，将 D 上的谓词映射为 C 上的谓词。`ExprPullback` 保持布尔代数结构：f*(p ∧ q) = f*(p) ∧ f*(q)，f*(¬p) = ¬f*(p)。

kont 建模代数效应（自由单子、处理器、折叠）。cove 建模余效应（余单子、需求、展开）。二者在挂起边界上互为范畴对偶：kont 问计算对世界做了什么，cove 述计算从世界需要什么。

`ReqExpr` 将布尔代数去函数化为标签联合 — 布尔签名函子的始代数 — 使需求结构成为可检查的数据而非不透明的闭包 (Reynolds 1972)。

## References

- Uustalu, T. & Vene, V. "Comonadic Notions of Computation." *CMCS 2008*, pp. 263–284. https://doi.org/10.1016/j.entcs.2008.05.029
- Petricek, T., Orchard, D. & Mycroft, A. "Coeffects: A Calculus of Context-Dependent Computation." *ICFP 2014*, pp. 123–135. https://doi.org/10.1145/2628136.2628160
- Reynolds, J.C. "Definitional Interpreters for Higher-Order Programming Languages." *ACM '72*, pp. 717–740. https://doi.org/10.1145/800194.805852

## 平台支持

`cove` 是该模块中的纯 Go 包。它没有平台特定文件或 build tag；模块要求仅为 Go 1.26+。

## 许可证

MIT 许可证。详见 [LICENSE](LICENSE)。

©2026 [Hayabusa Cloud Co., Ltd.](https://code.hybscloud.com)
