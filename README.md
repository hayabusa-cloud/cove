[![Go Reference](https://pkg.go.dev/badge/code.hybscloud.com/cove.svg)](https://pkg.go.dev/code.hybscloud.com/cove)
[![Benchmarks](https://github.com/hayabusa-cloud/cove/actions/workflows/benchmarks.yml/badge.svg)](https://github.com/hayabusa-cloud/cove/actions/workflows/benchmarks.yml)
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

Requires Go 1.27+.

## Core Types

| Type                                                | Purpose                                                         |
|-----------------------------------------------------|-----------------------------------------------------------------|
| `View[C, A]`                                        | Comonadic carrier: value A under ambient context C              |
| `SuspensionView[C, A]`                              | kont suspension paired with ambient context                     |
| `World`                                             | Kripke reading of an ambient context type                       |
| `StepIndex`                                         | Finite approximation level for step-indexed observation         |
| `Preorder[C]`                                       | Kripke world-extension relation `w <= w'` with local law checks |
| `Transition[C]`                                     | Concrete candidate edge between two Kripke worlds               |
| `Forcing[C]` / `ForcingExpr[C]`                     | Requirement plus world preorder with monotonicity checks        |
| `Relation[C, A]`                                    | Step-indexed Kripke relation over world, index, and value       |
| `IndexedView[C, A]` / `IndexedSuspensionView[C, A]` | Contextual observations paired with explicit step credit        |
| `Req[C]`                                            | Contravariant predicate over C (closure form): `func(C) bool`   |
| `ReqExpr[C]`                                        | Contravariant predicate over C (defunctionalized form)          |
| `Rule[C]` / `RuleExpr[C]`                           | Named predicate with diagnostic `Report`                        |
| `Checked[C, A]` / `CheckedExpr[C, A]`               | Value gated by a requirement                                    |
| `Guarded[C, A]` / `GuardedExpr[C, A]`               | Value gated by a named rule                                     |

`Ambient` constrains context type parameters (C, D). `World` gives the same structural constraint the Kripke vocabulary,
and `Focus` constrains value type parameters (A, B).

## Contextual Stepping

`StepWith` and `StepExprWith` evaluate a kont computation and pair its first suspension with ambient context. Each
suspension is affine: consume it exactly once, via `Resume`, `ResumeWith`, or `Discard`.

Use `StepWith(ctx, m)` to step a `Cont` directly. `Reify(m)` runs to the first return or operation. A completed result
is reusable. A pending result and its copies share one invocation. Stepping it twice panics, including through
`Reflect`. Reify the original reusable `Cont` again for each run.

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
sv = cove.MapContextSuspension(sv, func(r Runtime) Runtime {
	r.Budget += 4
	return r
})
sv = cove.WithContextSuspension(sv, Runtime{Budget: 16})
```

Once the computation completes, `sv.Suspension` becomes nil, but `sv.Ask()` still returns the carried context.

cove forwards kont's stepping classifier: a nil suspension identifies completion, and the returned value is the completed result. Nil is a valid result for a nilable type; an interface containing a typed nil retains its dynamic type. Suspended steps may also return the zero value of `A`, so inspect the suspension to distinguish completion. Carrier context is preserved.

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

A refused check leaves the original suspension unconsumed; retain that handle until a later check or explicit `Discard`.
Refresh context from its owner before checking, and protect the condition through the action it authorizes, including
callback reentry. `ResumeWith` runs its mapper after advancement, including completion. `ResumeTo` checks a supplied
world edge; it neither obtains fresh observations nor checks an arbitrary requirement. Revocable eligibility is not
persistent forcing, and `StepIndex` is logical credit, not time. Dispatch, continuation advancement and external resource
retirement have separate contracts; `Discard` does not settle I/O. The [temporal example](temporal_example_test.go)
demonstrates refusal, refresh and completion using a pure operation without external resources.

## Step-Indexed Kripke Evidence

`cove` can make the Kripke reading explicit without turning context into scheduler policy. A `Preorder[C]` defines world
extension and exposes `Extends`, `ReflexiveAt`, and `TransitiveAt` checks; `Transition[C]`, `CheckTransition`,
`CheckForcingTransition`, and `CheckInvariantTransition` check concrete world edges; `Forcing` pairs a requirement with
that preorder; and `Relation` indexes semantic validity by world, finite `StepIndex`, and value.

Use `DiscreteWorlds` for equality-only worlds and `TotalWorlds` for a preorder that relates every world pair.
`ForceExpr` mirrors `Force` for `ReqExpr`; `CheckRelation`, `WeakenIndexedView`, and `WeakenIndexedSuspension` make
relation checks and step-credit weakening explicit.

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

For operational prefixes, `StepWithIndex` and `StepExprWithIndex` pair a `SuspensionView` with explicit fuel. Each
indexed `Resume` consumes one unit of step credit, making finite observations visibly decrease. Use `ResumeTo` when the
successor context must be checked as a Kripke world extension; `CheckCompletedRelation` checks the final relation at a
completed indexed frontier:

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

## Commands

`Cmd[C, A, B]` is the coKleisli arrow `View[C, A] -> B`. `Run` applies a command to a concrete `View`; `ExtractCmd` is
the identity command; `LiftCmd` lifts a focus-only map into the contextual world; `Compose` composes commands through
`Extend`, satisfying `Compose(g, f)(v) == g(Extend(v, f))`.

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

The **data form** (`ReqExpr`) stores the Boolean structure.

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

Use `Req` for a direct predicate and `ReqExpr` to preserve Boolean structure. Both can be built once. Recheck the guard
before acting when its inputs can change.

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

For fixed C and pure callbacks, `View[C, A]` has the environment-comonad structure described below. The carrier is the product C × A; `Extract` (ε) projects the value; `Duplicate` (δ) embeds the observation as the focused value:

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

| Package    | Owns                                                   | Categorical Aspect               |
|------------|--------------------------------------------------------|----------------------------------|
| `kont`     | Effect operations, handlers, suspension and resumption | Algebra / Free monad / Fold      |
| **`cove`** | **Ambient context across suspension boundaries**       | **Coalgebra / Comonad / Unfold** |
| `takt`     | Proactor dispatch, stepping and runner observation     | Comodel (handler dual)           |
| `uring`    | Kernel I/O: SQE/CQE, ring management, buffer rings     | —                                |
| `iox`      | Outcome algebra and error semantics                    | —                                |

## Formal Structure

For fixed C and pure, total callbacks, `View[C, A]` is an environment comonad (Uustalu & Vene 2008): carrier C × A, counit ε = π₂ (`Extract`), and comultiplication δ(c, a) = (c, (c, a)) (`Duplicate`). `Extend` implements coKleisli extension. The equations use observational equality and stable context; they do not require Go-comparable values or functions.

`Req[C]` denotes a predicate; `ReqExpr[C]` stores Boolean syntax with opaque function atoms. For pure predicates and projections, `Pullback` implements the contravariant map f* induced by f: C → D. `ExprPullback` preserves the Boolean structure: f*(p ∧ q) = f*(p) ∧ f*(q), f*(¬p) = ¬f*(p). It may apply the projection to each visited atom.

kont supplies effectful computations and resumptions; cove carries the context required to observe and check their boundaries. The caller acquires facts, supplies synchronization, and owns execution and resource lifetimes.

Composite requirements evaluate left to right with short circuiting. `All`, `Any`, `ExprAll`, and `ExprAny` retain multiple operands supplied as a slice; keep that storage unchanged while used. View copies share referenced storage. `ExprAtom` and `ReifyReq` require non-nil predicates.

`ReqExpr` represents Boolean composition as data while leaf predicates remain Go functions. Evaluation interprets that structure. `ReifyReq` wraps an opaque predicate as an atom; it does not extract syntax from a closure or specialize execution automatically.

## Practical Recipes

A complete contextual computation typically proceeds in three stages: declare requirements, gate values against them,
then observe the result under a concrete context.

```go
// 1. Declare a requirement on the ambient context.
type Caps struct{ CanSubmit, HasToken bool }
req := cove.All(
	func(c Caps) bool { return c.CanSubmit },
	func(c Caps) bool { return c.HasToken },
)

// 2. Gate a value on that requirement. This produces a Checked[Caps, T].
checked := cove.Guard(req, payload)

// 3. Check the requirement under a concrete context before using the value.
if view, ok := checked.IntoView(Caps{CanSubmit: true, HasToken: true}); ok {
	_ = view.Extract()
}
```

`Pullback` lets a caller adapt a requirement defined over one context so that it works over another (`Pullback(req, f)`
for `f: D → C`); `MapChecked` and `MapGuarded` transport gated values through value-level transformations without
re-checking the predicate. The combination `All` + `Pullback` + `Guard` is how downstream packages build typed
capability checks without coupling to a single context type.

## Performance Measurement

See the [benchmark guide](BENCHMARKS.md) for comparable workloads, measurement commands, and allocation boundaries in your environment.

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

`cove` is a pure Go package in this module. It has no platform-specific files or build tags; the module requirement is Go 1.27+.

## License

MIT License. See [LICENSE](LICENSE) for details.

©2026 [Hayabusa Cloud Co., Ltd.](https://code.hybscloud.com)
