[![Go Reference](https://pkg.go.dev/badge/code.hybscloud.com/cove.svg)](https://pkg.go.dev/code.hybscloud.com/cove)
[![Benchmarks](https://github.com/hayabusa-cloud/cove/actions/workflows/benchmarks.yml/badge.svg)](https://github.com/hayabusa-cloud/cove/actions/workflows/benchmarks.yml)
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
for sv.Suspension != nil && sv.Ask().Budget > 0 {
    result := dispatch(sv.Ask(), sv.Op())
    _, sv = sv.ResumeWith(result, func(r Runtime) Runtime {
        r.Budget--
        return r
    })
}
```

循环结束时，若 `sv.Suspension != nil`，则保留 `sv`，并在下次分派前补充预算。

本包是无策略的：只承载与检查上下文，从不调度、重试或分派。

## 安装

```bash
go get code.hybscloud.com/cove
```

需要 Go 1.27+。

## 核心类型

| 类型                                                  | 用途                              |
|-----------------------------------------------------|---------------------------------|
| `View[C, A]`                                        | 余单子载体：环境上下文 C 下的值 A             |
| `SuspensionView[C, A]`                              | 与环境上下文配对的 kont 挂起               |
| `World`                                             | 环境上下文类型的 Kripke 读法              |
| `StepIndex`                                         | 步进索引观测的有限近似层级                   |
| `Preorder[C]`                                       | Kripke 世界扩展关系 `w <= w'`，带局部定律检查 |
| `Transition[C]`                                     | 两个 Kripke 世界之间的具体候选边            |
| `Forcing[C]` / `ForcingExpr[C]`                     | 带世界预序与单调性检查的需求                  |
| `Relation[C, A]`                                    | 关于世界、索引和值的步进索引 Kripke 关系        |
| `IndexedView[C, A]` / `IndexedSuspensionView[C, A]` | 带显式步进额度的上下文观测                   |
| `Req[C]`                                            | C 上的逆变谓词（闭包形式）：`func(C) bool`   |
| `ReqExpr[C]`                                        | C 上的逆变谓词（去函数化形式）                |
| `Rule[C]` / `RuleExpr[C]`                           | 带诊断 `Report` 的命名谓词              |
| `Checked[C, A]` / `CheckedExpr[C, A]`               | 由需求门控的值                         |
| `Guarded[C, A]` / `GuardedExpr[C, A]`               | 由命名规则门控的值                       |

`Ambient` 约束上下文类型参数 (C, D)。`World` 为同一结构约束提供 Kripke 词汇，`Focus` 约束值类型参数 (A, B)。

## 上下文步进

`StepWith` 与 `StepExprWith` 对 kont 计算求值，并将其第一个挂起与环境上下文配对。返回的 `SuspensionView` 的各个副本共享同一个
`*kont.Suspension` 指针。等待应答时可以保留视图，用 `Resume` 或 `ResumeWith` 恢复一次，或用 `Discard` 放弃。

用 `StepWith(ctx, m)` 直接推进 `Cont`。`Reify(m)` 将计算推进到首次返回或产生操作。已完成的结果可以复用。
挂起的结果及其副本共享同一次调用。对同一结果推进两次会触发 panic，经由 `Reflect` 也是如此。
若要重复执行，每次都应对原来可复用的 `Cont` 重新调用 `Reify`。

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

使用 `Resume` 或 `ResumeWith` 返回的视图 `sv`。计算完成时，该视图的 `Suspension` 字段为 nil，`sv.Ask()` 返回视图携带的上下文。

cove 直接使用 kont 的步进分类：nil 挂起指针表示计算完成，返回值是完成结果。对于允许 nil 的类型，nil 是有效结果；接口中带类型的 nil 会保留其动态类型。挂起步骤也可能返回 `A` 的零值，因此应检查挂起指针来判断是否完成。所携带的上下文保持不变。

`ObserveSuspension` 为已有的挂起附加上下文。下面的驱动在每次分派前检查当前上下文，检查未通过时丢弃挂起。
这里的 `dispatch` 同步返回当前挂起操作的应答。

```go
canRun := func(r Runtime) bool { return r.Budget > 0 }
val, susp := cove.Step(m)
for susp != nil {
    sv, ok := cove.CheckSuspension(currentContext(), susp, canRun)
    if !ok {
        susp.Discard()
        return
    }
    val, susp = susp.Resume(dispatch(sv.Ask(), sv.Op()))
}
complete(val)
```

若使用表达式需求，先构造一次 `req := cove.ExprAtom(canRun)`，再在循环内调用
`cove.CheckSuspensionExpr(currentContext(), susp, req)`。

`Step` 和 `StepExpr` 重新导出 kont 的评估器，不绑定上下文。

检查未通过时，所有者可以保留原挂起，稍后再检查，也可以调用 `Discard`。遇到 `iox.ErrWouldBlock` 时，
`takt.AdvanceSuspension` 和 `sess.Advance` 都返回同一个挂起句柄。重试前应重新检查当前上下文。
从检查到执行获准的操作，应确保条件持续成立，包括回调重入期间。`ResumeWith` 在推进之后执行映射，完成时也如此。`ResumeTo` 只检查传入的世界边，不获取新观测，也不检查任意需求。
可撤销的资格不等于持久 forcing；`StepIndex` 是逻辑额度，不是时间。派发、续体推进和外部资源释放各有独立契约；`Discard` 不会完成 I/O。

用实现 `tempo.Residual` 的具体类型表示各个独立的时序条件，再用 `tempo.Both` 组合。调用方对匹配的事件执行
`state = state.Step(event)`，并将 `state.Observe()` 传给 `CheckSuspension`。就绪事件和前置操作的完成回执可以按任意顺序到达。
恢复计算时，当前代次必须仍处于就绪状态，并且已收到属于该代次的完成回执。
[时序示例](temporal_example_test.go) · [tempo](https://code.hybscloud.com/tempo)

## 步进索引 Kripke 证据

`cove` 可以显式表达 Kripke 结构而不引入调度策略。`Preorder[C]` 定义世界扩展，并提供 `Extends`、`ReflexiveAt`、`TransitiveAt`
局部检查；`Transition[C]`、`CheckTransition`、`CheckForcingTransition` 与 `CheckInvariantTransition` 检查具体世界边；
`Forcing` 将需求与该预序配对，`Relation` 按世界、有限 `StepIndex` 与值索引语义有效性。

当世界值可以用 Go 的 `==` 比较且每个值都等于自身时，使用 `DiscreteWorlds`。`TotalWorlds` 将任意两个世界关联起来。
`ForceExpr` 是 `Force` 在 `ReqExpr` 上的对应形式。`CheckRelation`、`WeakenIndexedView` 与 `WeakenIndexedSuspension`
显式表达关系检查和步进额度弱化。

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

`StepWithIndex` 和 `StepExprWithIndex` 返回第一个结果或操作时保留全部初始步数。每次恢复执行消耗一步。
`ResumeTo` 还会检查后继世界是否扩展了当前世界。如果剩余步数为零，这个循环会在分派操作前丢弃待恢复的续体。
计算完成后，`CheckCompletedRelation` 根据最终世界和剩余步数检查返回值。

```go
val, sv := cove.StepExprWithIndex(RuntimeWorld{Epoch: 2}, 2, expr)
for !sv.Completed() {
    if sv.Index.IsZero() {
        sv.Discard()
        return
    }
    result := dispatch(sv.Ask(), sv.Op())
    next := sv.Ask()
    next.Epoch++
    val, sv = sv.ResumeTo(leq, result, next)
}
_ = cove.CheckCompletedRelation(sv, val, rel)
```

## 命令

`Cmd[C, A, B]` 是上下文命令 `View[C, A] -> B`。`Run` 将命令应用到具体的 `View`；`ExtractCmd` 是恒等命令；`LiftCmd`
把仅作用于焦点值的映射提升到上下文世界。将 `Compose(g, f)` 应用于视图 `v` 时，会计算 `g(Extend(v, f))`。

```go
cmd := cove.Compose(
    func(v cove.View[Runtime, int]) string {
        return fmt.Sprintf("budget=%d value=%d", v.Ask().Budget, v.Extract())
    },
    cove.LiftCmd[Runtime](func(n int) int { return n + 1 }),
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

**数据形式**（`ReqExpr`）保留布尔结构。

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

直接判断用 `Req`，需要保留布尔结构时用 `ReqExpr`。两者都可以构建一次并复用。如果输入可能变化，应在行动前重新判断。

如果 `delivered` 检查是否收到了回执，在投递结果尚未确定时，`Not(delivered)` 也为真。只允许在确认被拒后重试时，需要单独的谓词。

`ReflectReq(expr)` 让表达式可以作为 `Req` 求值。`ReifyReq(req)` 把非 nil 谓词包装为单个 `ExprAtom`。
`Need(ctx, nil)` 返回真。需要无约束的表达式时使用 `ExprTrue[C]()`。`NeedExpr(ctx, ReifyReq[C](nil))` 会触发 panic。

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

对于固定的 C 和纯回调，`View[C, A]` 具有下文描述的环境余单子结构。载体为积 C × A，`Extract`（ε）投影值，`Duplicate`（δ）将观察嵌入为焦点值：

```go
v := cove.Observe(ctx, value)
v.Ask()       // 环境上下文 (π₁)
v.Extract()   // 聚焦值 (ε = π₂)
```

变换：`Map`（在 `A` 上的函子提升）、`MapContext`（在 `C` 上的函子提升）、`Replace`（替换值）、`WithContext`（替换上下文）。

`Duplicate` 和 `Extend` 的定律假设上下文稳定，回调是处处有定义的纯函数。`≈` 表示观测等价。[Go 测试](view_test.go)

| 定律 | 等式 |
| --- | --- |
| 余单位律 | `Duplicate(v).Extract() ≈ v` |
| 扩展恒等律 | `Extend(v, ExtractCmd) ≈ v` |
| 结合律 | `Extend(Extend(v, f), g) ≈ Extend(v, Compose(g, f))` |

`Extend` 是上下文扩展算子：给定 `f: W(C, A) → B`，在保持环境上下文的同时将 `f` 提升为 `W(C, A) → W(C, B)`。与
`g: W(C, B) → D` 诱导出的命令组合为 `g ∘ Extend(f)`。

## 生态系统定位

| 包          | 职责                     | 范畴性质               |
|------------|------------------------|--------------------|
| `kont`     | 效应操作、处理器、挂起与恢复         | 代数 / 自由单子 / 折叠     |
| `tempo`    | 时序状态转移与观测                 | 摩尔机 / 同步积           |
| **`cove`** | **跨挂起边界的环境上下文**        | **余代数 / 余单子 / 展开** |
| `takt`     | 主动器分派、步进与运行器观测         | 余模型（处理器对偶）         |
| `uring`    | 内核 I/O：SQE/CQE、环管理、缓冲环 | —                  |
| `iox`      | 结果代数与错误语义              | —                  |

## 形式结构

在 C 固定、回调为纯全函数的条件下，`View[C, A]` 是环境余单子 (Uustalu & Vene 2008)：载体为 C × A，余单位 ε = π₂（`Extract`），余乘法 δ(c, a) = (c, (c, a))（`Duplicate`）。`Extend` 实现余 Kleisli 扩展。等式表示上下文稳定时的观测等价，不要求值或函数在 Go 中可比较。

`Req[C]` 表示谓词，`ReqExpr[C]` 保存以不透明函数为原子的布尔语法。对于纯谓词和纯投影，`Pullback` 实现由 f: C → D 诱导的逆变映射 f*。`ExprPullback` 保持布尔结构：f*(p ∧ q) = f*(p) ∧ f*(q)，f*(¬p) = ¬f*(p)。求值时可能对每个访问到的原子调用投影。

kont 提供带效应的计算与恢复操作；cove 携带观测和检查这些边界所需的上下文。事实获取、同步控制、执行及资源生命周期由调用者负责。

组合需求按从左到右的顺序短路求值。`All`、`Any`、`ExprAll` 和 `ExprAny` 会保留通过切片传入的多个操作数；使用期间应保持该存储不变。View 的副本共享所引用的存储。`ExprAtom` 和 `ReifyReq` 要求非 nil 谓词。

`ReqExpr` 用数据表示布尔组合，叶子谓词仍是 Go 函数，求值过程解释执行该结构。`ReifyReq` 将不透明谓词包装为原子，不会从闭包提取语法或自动特化执行。

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

// 3. 在具体上下文中检查需求，然后使用该值。
if view, ok := checked.IntoView(Caps{CanSubmit: true, HasToken: true}); ok {
	_ = view.Extract()
}
```

`Pullback` 让调用者把定义在某个上下文上的需求适配到另一个上下文（`Pullback(req, f)`，其中 `f: D → C`）；`MapChecked` 与
`MapGuarded` 可在不重新检查谓词的前提下，沿值层面的变换搬运被守护的值。`All` + `Pullback` + `Guard`
这一组合，正是下游包用以构建带类型的能力检查、又不与任一具体上下文类型耦合的方式。

## 性能测量

[基准测试指南](BENCHMARKS.md)提供可比较的工作负载、测量命令和内存分配测量范围，供您在自己的环境中测量。

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

`cove` 是该模块中的纯 Go 包。它没有平台特定文件或 build tag；模块要求仅为 Go 1.27+。

## 许可证

MIT 许可证。详见 [LICENSE](LICENSE)。

©2026 [Hayabusa Cloud Co., Ltd.](https://code.hybscloud.com)
