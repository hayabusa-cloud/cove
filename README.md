[![Go Reference](https://pkg.go.dev/badge/code.hybscloud.com/cove.svg)](https://pkg.go.dev/code.hybscloud.com/cove)
[![Go Report Card](https://goreportcard.com/badge/github.com/hayabusa-cloud/cove)](https://goreportcard.com/report/github.com/hayabusa-cloud/cove)
[![Coverage Status](https://codecov.io/gh/hayabusa-cloud/cove/graph/badge.svg)](https://codecov.io/gh/hayabusa-cloud/cove)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

**English** | [简体中文](README.zh-CN.md) | [Español](README.es.md) | [日本語](README.ja.md) | [Français](README.fr.md)

# cove

Context layer for carrying explicit ambient context across [kont](https://code.hybscloud.com/kont) suspension boundaries
in Go.

## Overview

When an external runtime such as a proactor, event loop, or dispatcher steps through kont suspensions one at a time,
each suspended operation may need state that lives outside the operation itself: dispatch budget, ring capabilities,
protocol phase, buffer-group validity. Without explicit context carriers, that state ends up in ad-hoc side maps or
implicit globals that break composition.

cove pairs ambient context with values and suspensions so the context travels across asynchronous stepping boundaries as
typed, composable data.

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

The package is policy-free: it carries and checks context, but never schedules, retries, or dispatches.

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

`StepWith` and `StepExprWith` evaluate a kont computation and pair its first suspension with ambient context. Each
suspension is affine: consume it exactly once, via `Resume`, `ResumeWith`, or `Discard`.

```go
val, sv := cove.StepExprWith(ctx, expr)
for sv.Suspension != nil {
    op := sv.Op()
    result := handle(op)
    val, sv = sv.Resume(result)
}
```

`ResumeWith` evolves the carried context between steps. For example, it can decrement budget, update capabilities, or
advance protocol phase:

```go
val, sv = sv.ResumeWith(result, func(r Runtime) Runtime {
    r.Budget--
    return r
})
```

`MapContextSuspension` and `WithContextSuspension` transport or replace the context carried on an already observed
suspension, without changing the current suspension frontier:

```go
sv = cove.MapContextSuspension(sv, func (r Runtime) Runtime {
r.Budget += 4
return r
})
sv = cove.WithContextSuspension(sv, Runtime{Budget: 16})
```

Once the computation completes, `sv.Suspension` becomes nil, but `sv.Ask()` still returns the carried context.

cove forwards kont's stepping classifier, so `Step`, `StepExpr`, `StepWith`, `StepExprWith`, `Resume`, and `ResumeWith`
inherit kont's nil-completion convention: a nil completed value denotes completion with the zero value of `A`.
Computations whose result type is a pointer or interface therefore cannot use nil as a meaningful completed value; wrap
nil in an explicit sum or witness type when that distinction matters. Carrier context is unaffected.

`ObserveSuspension` attaches ambient context to an existing `kont.Suspension` without performing any requirement check.
`CheckSuspension` and `CheckSuspensionExpr` provide gated forms:

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

## Commands

`Cmd[C, A, B]` is the coKleisli arrow `View[C, A] -> B`. `Run` applies a command to a concrete `View`; `ExtractCmd` is
the identity command; `LiftCmd` lifts a focus-only map into the contextual world; `Compose` composes commands through
`Extend`, satisfying `Compose(g, f)(v) == g(Extend(v, f))`.

```go
cmd := cove.Compose(
func (v cove.View[Runtime, int]) string {
return fmt.Sprintf("budget=%d value=%d", v.Ask().Budget, v.Extract())
},
cove.LiftCmd(func (n int) int { return n + 1 }),
)
out := cove.Run(cove.Observe(Runtime{Budget: 8}, 41), cmd)
_ = out // "budget=8 value=42"
```

## Requirements

Requirements are predicates over the ambient context, available in both closure and data form. Both forms support `All`,
`Any`, and `Not`, with `True` and `False` as the neutral elements of conjunction and disjunction.

**Closure form** (`Req`): direct and concise.

```go
req := cove.All(
    cove.Req[Runtime](func(r Runtime) bool { return r.Budget > 0 }),
    cove.Req[Runtime](func(r Runtime) bool { return r.CanDispatch }),
)
ok := cove.Need(runtime, req)
```

**Data form** (`ReqExpr`): composable Boolean structure that avoids closure allocation during composition.

```go
expr := cove.ExprAll(
    cove.ExprAtom(func(r Runtime) bool { return r.Budget > 0 }),
    cove.ExprAtom(func(r Runtime) bool { return r.CanDispatch }),
)
ok := cove.NeedExpr(runtime, expr)
```

Combinators: `All`/`ExprAll` (conjunction ∧), `Any`/`ExprAny` (disjunction ∨), `Not`/`ExprNot` (negation ¬), `True`/`ExprTrue` (⊤), `False`/`ExprFalse` (⊥).

`Pullback` and `ExprPullback` transport requirements contravariantly along a context projection. Given `f: C → D`,
`Pullback(req, f)` maps `Req[D]` to `Req[C]`. `ExprPullback` preserves the Boolean structure and distributes over
`ExprNot`, `ExprAll`, and `ExprAny`.

### Choosing Req vs ReqExpr

Use `Req` for ad-hoc, one-off predicates. Use `ReqExpr` when composing many requirements, or when closure allocation at
construction time matters.

Bridge helpers keep the two forms aligned: `ReifyReq` is a lossy quotation helper that wraps a closure-form requirement
as an `ExprAtom`, so `ReifyReq(nil)` is invalid and panics when evaluated; `ReflectReq` evaluates an Expr-form
requirement through a closure rather than recovering its original structure.

## Gated Values

`Guard` pairs a value with a requirement; `IntoView` yields a `View` only when the ambient context satisfies that
requirement:

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

`CheckRules` and `CheckRulesExpr` evaluate several rules in order, stopping at the first failure:

```go
report := cove.CheckRules(runtime, budgetRule, permRule)
if !report.OK() {
    // report.Failed names the first rule that did not hold
}
```

The Expr-form diagnostic path has the same shape through `CheckRulesExpr` and `GuardRuleExpr`:

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

Expr-form equivalents: `GuardExpr`, `GuardRuleExpr`, `CheckedExpr`, `GuardedExpr`. Map and pullback helpers:
`MapChecked`, `MapGuarded`, `PullbackChecked`, `PullbackGuarded` (and their Expr variants).

## View Operations

`View[C, A]` is a comonad over the category of Go types. The carrier is the product C × A; `Extract` (ε) projects the value; `Duplicate` (δ) embeds the observation as the focused value:

```go
v := cove.Observe(ctx, value)
v.Ask()       // ambient context  (π₁)
v.Extract()   // value in focus   (ε = π₂)
```

Transformations: `Map` (functorial lift on `A`), `MapContext` (functorial lift on `C`), `Replace` (substitute the
value), `WithContext` (substitute the context).

`Duplicate` and `Extend` satisfy the comonad laws:

```go
// Counit law:  ε ∘ δ = id
cove.Duplicate(v).Extract() == v

// Extension identity:  extend(ε) = id
cove.Extend(v, func(w cove.View[C, A]) A { return w.Extract() }) == v

// Associativity:  extend(f) ∘ extend(g) = extend(f ∘ extend(g))
```

`Extend` is the contextual extension operator: given `f: W(C, A) → B`, it lifts `f` to `W(C, A) → W(C, B)` while
preserving the ambient context. The induced command composition with `g: W(C, B) → D` is `g ∘ Extend(f)`.

## Ecosystem Position

| Package | Owns | Categorical Aspect |
|---------|------|--------------------|
| `kont` | Effect operations, handlers, suspension and resumption | Algebra / Free monad / Fold |
| **`cove`** | **Ambient context across suspension boundaries** | **Coalgebra / Comonad / Unfold** |
| `takt` | Proactor dispatch, outcome classification, event loop | Comodel (handler dual) |
| `uring` | Kernel I/O: SQE/CQE, ring management, buffer rings | — |
| `iox` | Outcome algebra and error semantics | — |

## Formal Structure

`View[C, A]` is an environment comonad (Uustalu & Vene 2008) with carrier C × A, counit ε = π₂ (Extract), and
comultiplication δ(c, a) = (c, (c, a)) (Duplicate). `Extend` implements contextual extension (coKleisli extension in the
categorical literature).

`Req[C]` and `ReqExpr[C]` are objects in the Boolean algebra of predicates over C. `Pullback` implements the contravariant functor f* induced by a morphism f: C → D, mapping predicates on D to predicates on C. `ExprPullback` preserves the Boolean algebra structure: f*(p ∧ q) = f*(p) ∧ f*(q), f*(¬p) = ¬f*(p).

kont models algebraic effects (free monad, handlers, fold); cove models coeffects (comonad, requirements, unfold). The
two are categorical duals at the suspension boundary: kont describes what the computation does to the world, whereas
cove states what the computation needs from the world.

Read through the lens of modal effects, the carrier shape can be read as naming what happens to ambient context: kont (
C → D) replaces it, cove (W(C, A) → B) handles it relatively, and pure pass-through preserves it. Effect structure rides
on the carrier, not on a function colour, which keeps cove policy-free.

`ReqExpr` defunctionalizes the Boolean algebra into a tagged union, namely the initial algebra of the Boolean signature
functor, so that the structure of a requirement is inspectable data rather than an opaque closure (Reynolds 1972).

## Practical Recipes

A complete contextual computation typically proceeds in three stages: declare requirements, gate values against them,
then observe the result under a concrete context.

```go
// 1. Declare a requirement on the ambient context.
type Caps struct{ CanSubmit, HasToken bool }
req := cove.All(
func (c Caps) bool { return c.CanSubmit },
func (c Caps) bool { return c.HasToken },
)

// 2. Gate a value on that requirement. This produces a Checked[Caps, T].
checked := cove.Guard(req, payload)

// 3. Reify into an Expr to step under different ambient contexts.
expr := cove.ReifyReq(req)
ok := cove.NeedExpr(Caps{CanSubmit: true, HasToken: true}, expr)
_ = checked
_ = ok
```

`Pullback` lets a caller adapt a requirement defined over one context so that it works over another (`Pullback(req, f)`
for `f: D → C`); `MapChecked` and `MapGuarded` transport gated values through value-level transformations without
re-checking the predicate. The combination `All` + `Pullback` + `Guard` is how downstream packages build typed
capability checks without coupling to a single context type.

## References

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

## Platform Support

`cove` is a pure Go package in this module. It has no platform-specific files or build tags; the module requirement is Go 1.26+.

## License

MIT License. See [LICENSE](LICENSE) for details.

©2026 [Hayabusa Cloud Co., Ltd.](https://code.hybscloud.com)
