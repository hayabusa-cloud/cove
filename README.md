[![Go Reference](https://pkg.go.dev/badge/code.hybscloud.com/cove.svg)](https://pkg.go.dev/code.hybscloud.com/cove)
[![Go Report Card](https://goreportcard.com/badge/github.com/hayabusa-cloud/cove)](https://goreportcard.com/report/github.com/hayabusa-cloud/cove)
[![Coverage Status](https://codecov.io/gh/hayabusa-cloud/cove/graph/badge.svg)](https://codecov.io/gh/hayabusa-cloud/cove)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

**English** | [简体中文](README.zh-CN.md) | [Español](README.es.md) | [日本語](README.ja.md) | [Français](README.fr.md)

# cove

Coalgebraic context layer for [kont](https://code.hybscloud.com/kont) suspensions in Go, the coeffect dual of kont's algebraic effects.

## Overview

When an external runtime, such as a proactor, event loop, or dispatcher steps through kont suspensions one at a time, each suspended operation may need state that lives outside the operation itself: dispatch budget, ring capabilities, protocol phase, buffer-group validity. Without explicit context carriers, this state ends up in ad-hoc side maps or implicit globals that break composition.

cove pairs ambient context with values and suspensions so the context travels through async stepping boundaries as typed, composable data.

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

The package is policy-free: it carries and checks context but never schedules, retries, or dispatches.

## Installation

```bash
go get code.hybscloud.com/cove
```

Requires Go 1.26+.

## Core Types

| Type | Purpose                                                       |
|------|---------------------------------------------------------------|
| `View[C, A]` | Comonadic carrier: value A under ambient context C            |
| `SuspensionView[C, A]` | kont suspension paired with ambient context                   |
| `Req[C]` | Contravariant predicate over C (closure form): `func(C) bool` |
| `ReqExpr[C]` | Contravariant predicate over C (defunctionalized form)        |
| `Rule[C]` / `RuleExpr[C]` | Named predicate with diagnostic `Report`                      |
| `Checked[C, A]` / `CheckedExpr[C, A]` | Value gated by a requirement                                  |
| `Guarded[C, A]` / `GuardedExpr[C, A]` | Value gated by a named rule                                   |

`Ambient` constrains context type parameters (C, D). `Focus` constrains value type parameters (A, B).

## Contextual Stepping

`StepWith` and `StepExprWith` evaluate a kont computation and pair the first suspension with ambient context. Each suspension is affine: consume it exactly once via `Resume`, `ResumeWith`, or `Discard`.

```go
val, sv := cove.StepExprWith(ctx, expr)
for sv.Suspension != nil {
    op := sv.Op()
    result := handle(op)
    val, sv = sv.Resume(result)
}
```

`ResumeWith` evolves context between steps, for example by decrementing budget, updating capabilities, or advancing protocol phase:

```go
val, sv = sv.ResumeWith(result, func(r Runtime) Runtime {
    r.Budget--
    return r
})
```

`ObserveSuspension` attaches ambient context to an existing `kont.Suspension` without a requirement check. `CheckSuspension` and `CheckSuspensionExpr` add gated forms:

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

`Step` and `StepExpr` re-export kont's evaluators without context binding.

## Requirements

Requirements are predicates over context, available in closure and data forms. Both support `All`/`Any`/`Not`, with `True` and `False` as the neutral elements for conjunction and disjunction.

**Closure form** (`Req`) — direct and concise:

```go
req := cove.All(
    cove.Req[Runtime](func(r Runtime) bool { return r.Budget > 0 }),
    cove.Req[Runtime](func(r Runtime) bool { return r.CanDispatch }),
)
ok := cove.Need(runtime, req)
```

**Data form** (`ReqExpr`) — composable Boolean structure, avoids closure allocation on composition:

```go
expr := cove.ExprAll(
    cove.ExprAtom(func(r Runtime) bool { return r.Budget > 0 }),
    cove.ExprAtom(func(r Runtime) bool { return r.CanDispatch }),
)
ok := cove.NeedExpr(runtime, expr)
```

Combinators: `All`/`ExprAll` (conjunction ∧), `Any`/`ExprAny` (disjunction ∨), `Not`/`ExprNot` (negation ¬), `True`/`ExprTrue` (⊤), `False`/`ExprFalse` (⊥).

`Pullback` and `ExprPullback` transport requirements contravariantly along a context projection. Given f: C → D, `Pullback(req, f)` maps `Req[D]` to `Req[C]`. `ExprPullback` preserves Boolean structure and distributes over `ExprNot`, `ExprAll`, and `ExprAny`.

### Choosing Req vs ReqExpr

Use `Req` for ad-hoc one-off predicates. Use `ReqExpr` when composing many requirements or when closure allocation at construction time matters.

Bridge helpers keep the two forms aligned: `ReifyReq` wraps a closure-form requirement as `ExprAtom`, so `ReifyReq(nil)` is invalid and panics when evaluated; `ReflectReq` evaluates an Expr-world requirement through a closure.

## Gated Values

`Guard` pairs a value with a requirement; `IntoView` yields a `View` only when the context satisfies it:

```go
checked := cove.Guard(canDispatch, payload)
if view, ok := checked.IntoView(runtime); ok {
    _ = view.Extract()
}
```

`GuardRule` adds diagnostics via named rules and `Report`:

```go
rule := cove.Require("budget", func(r Runtime) bool { return r.Budget > 0 })
guarded := cove.GuardRule(rule, payload)
if view, report := guarded.IntoView(runtime); report.OK() {
    _ = view.Extract()
}
```

`CheckRules` and `CheckRulesExpr` evaluate multiple rules, stopping at the first failure:

```go
report := cove.CheckRules(runtime, budgetRule, permRule)
if !report.OK() {
    // report.Failed names the first rule that did not hold
}
```

Expr-world diagnostics follow the same shape with `CheckRulesExpr` and `GuardRuleExpr`:

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

Expr-world equivalents: `GuardExpr`, `GuardRuleExpr`, `CheckedExpr`, `GuardedExpr`. Map and pullback helpers: `MapChecked`, `MapGuarded`, `PullbackChecked`, `PullbackGuarded` (and their Expr variants).

## View Operations

`View[C, A]` is a comonad over the category of Go types. The carrier is the product C × A; `Extract` (ε) projects the value; `Duplicate` (δ) embeds the observation as the focused value:

```go
v := cove.Observe(ctx, value)
v.Ask()       // ambient context  (π₁)
v.Extract()   // value in focus   (ε = π₂)
```

Transformations: `Map` (functorial lift on A), `MapContext` (functorial lift on C), `Replace` (substitute value), `WithContext` (substitute context).

`Duplicate` and `Extend` satisfy the comonad laws:

```go
// Counit law:  ε ∘ δ = id
cove.Duplicate(v).Extract() == v

// CoKleisli identity:  extend(ε) = id
cove.Extend(v, func(w cove.View[C, A]) A { return w.Extract() }) == v

// Associativity:  extend(f) ∘ extend(g) = extend(f ∘ extend(g))
```

`Extend` is coKleisli extension: given f: W(C,A) → B, it lifts f to W(C,A) → W(C,B) while preserving ambient context. The induced coKleisli composition with g: W(C,B) → D is g ∘ Extend(f).

## Ecosystem Position

| Package | Owns | Categorical Aspect |
|---------|------|--------------------|
| `kont` | Effect operations, handlers, suspension and resumption | Algebra / Free monad / Fold |
| **`cove`** | **Ambient context across suspension boundaries** | **Coalgebra / Comonad / Unfold** |
| `takt` | Proactor dispatch, outcome classification, event loop | Comodel (handler dual) |
| `uring` | Kernel I/O: SQE/CQE, ring management, buffer rings | — |
| `iox` | Outcome algebra and error semantics | — |

## Formal Structure

`View[C, A]` is an environment comonad (Uustalu & Vene 2008) with carrier C × A, counit ε = π₂ (Extract), and comultiplication δ(c, a) = (c, (c, a)) (Duplicate). `Extend` implements coKleisli extension.

`Req[C]` and `ReqExpr[C]` are objects in the Boolean algebra of predicates over C. `Pullback` implements the contravariant functor f* induced by a morphism f: C → D, mapping predicates on D to predicates on C. `ExprPullback` preserves the Boolean algebra structure: f*(p ∧ q) = f*(p) ∧ f*(q), f*(¬p) = ¬f*(p).

kont models algebraic effects (free monad, handlers, fold). cove models coeffects (comonad, requirements, unfold). The two are categorical duals at the suspension boundary: kont asks what the computation does to the world; cove states what the computation needs from the world.

`ReqExpr` defunctionalizes the Boolean algebra into a tagged union — the initial algebra of the Boolean signature functor — so that requirement structure is inspectable data rather than opaque closures (Reynolds 1972).

## References

- Uustalu, T. & Vene, V. "Comonadic Notions of Computation." *CMCS 2008*, pp. 263–284. https://doi.org/10.1016/j.entcs.2008.05.029
- Petricek, T., Orchard, D. & Mycroft, A. "Coeffects: A Calculus of Context-Dependent Computation." *ICFP 2014*, pp. 123–135. https://doi.org/10.1145/2628136.2628160
- Reynolds, J.C. "Definitional Interpreters for Higher-Order Programming Languages." *ACM '72*, pp. 717–740. https://doi.org/10.1145/800194.805852

## Platform Support

`cove` is a pure Go package in this module. It has no platform-specific files or build tags; the module requirement is Go 1.26+.

## License

MIT License. See [LICENSE](LICENSE) for details.

©2026 [Hayabusa Cloud Co., Ltd.](https://code.hybscloud.com)
