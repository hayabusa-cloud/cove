// ©Hayabusa Cloud Co., Ltd. 2026. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// Package cove carries explicit ambient context across [kont] suspension
// boundaries.
//
// When a runtime advances a [kont] computation one suspension at a time, each
// suspended operation may depend on state that lives outside the operation
// itself, such as dispatch budget, ring capabilities, protocol phase, or
// buffer-group validity. cove pairs that context with values and suspensions
// so it stays explicit across asynchronous boundaries instead of travelling
// through hidden globals or ad-hoc side maps.
//
// Read through the lens of modal effects, the carrier shape can be read as
// naming what happens to ambient context: kont (C → D) replaces it, cove
// (W(C, A) → B) handles it relatively, and pure pass-through preserves it.
// Effect structure rides on the carrier, not on a function colour, which keeps
// cove policy-free.
//
// Responsibilities stay split by layer:
//
//   - [code.hybscloud.com/iox] classifies outcome and progress evidence
//   - [kont] defines suspension and resumption shape
//   - cove carries and checks explicit context
//   - [code.hybscloud.com/takt] advances execution without taking ownership of that context
//
// The package is policy-free: it carries and checks context but never
// schedules, retries, or dispatches. Kernel mechanics belong to uring, and
// semantic outcome branching belongs to iox, including
// [code.hybscloud.com/iox.Classify]. The join point with takt is structural:
// a contextual carrier exposes the current suspended operation and accepts
// the matching resumption without transferring ownership of the context to
// the runner.
//
// Carriers:
//
//   - [View[C, A]], a focused value paired with ambient context
//   - [Cmd[C, A, B]], a contextual command over [View]
//   - [SuspensionView[C, A]], a [kont.Suspension] paired with ambient context
//   - [World], [StepIndex], [Preorder[C]], [Transition[C]], [Forcing[C]],
//     [ForcingExpr[C]], [Indexed[C, A]], and [Relation[C, A]], step-indexed Kripke evidence over ambient worlds
//   - [IndexedView[C, A]] and [IndexedSuspensionView[C, A]], contextual
//     observations paired with explicit finite step credit
//   - [Req[C]] / [ReqExpr[C]], context predicates in closure and data forms
//   - [Rule[C]] / [RuleExpr[C]] with [Report], named predicates for diagnostics
//   - [Checked[C, A]] / [CheckedExpr[C, A]], requirement-gated values
//   - [Guarded[C, A]] / [GuardedExpr[C, A]], rule-gated values
//
// Commands:
//
//   - [ExtractCmd] returns the focused value as the identity contextual command
//   - [LiftCmd] lifts a focus-only transform into a contextual command
//   - [Compose] composes contextual commands through [Extend]
//   - [Run] applies a command to a concrete [View]
//
// Contextual suspension boundary:
//
//   - [StepWith] / [StepExprWith] run a kont computation and pair its first suspension with context
//   - [StepWithIndex] / [StepExprWithIndex] do the same with explicit step credit
//   - [MapContextSuspension] / [WithContextSuspension] map or replace carried context explicitly
//   - [SuspensionView.Op] / [SuspensionView.Resume] expose the structural join point consumed by takt
//   - [SuspensionView.ResumeWith] advances a suspension and evolves context for the next step
//   - [IndexedSuspensionView.ResumeWith] additionally decrements the step index
//   - [IndexedSuspensionView.ResumeTo] decrements the index and checks monotone world extension
//   - [CheckSuspension] / [CheckSuspensionExpr] gate contextualization on a requirement
//
// Kripke structure:
//
//   - [World] names the Kripke reading of ambient context
//   - [Preorder[C]] names world extension, w <= w'
//   - [DiscreteWorlds] and [TotalWorlds] provide equality-only and total preorders
//   - [Extends] is the package-level world-extension helper
//   - [Preorder.Extends], [Preorder.ReflexiveAt], and [Preorder.TransitiveAt] expose concrete preorder law checks
//   - [Transition[C]], [CheckTransition], [CheckForcingTransition], and [CheckInvariantTransition] check concrete world edges
//   - [Force] / [ForceExpr] construct [Forcing[C]] and [ForcingExpr[C]] evidence with monotonicity checks
//   - [Relation[C, A]] is indexed by world, [StepIndex], and value
//   - [Later] implements the guarded later modality: index zero is trivial; index n+1 checks at n
//   - [CheckRelation], [WeakenIndexedView], and [WeakenIndexedSuspension] check indexed evidence or weaken finite credit
//   - [CheckCompletedRelation] checks the final finite-trace relation at a completed indexed frontier
//
// Nil-completion convention: cove forwards [kont]'s stepping classifier, so
// [Step], [StepExpr], [StepWith], [StepExprWith], [SuspensionView.Resume], and
// [SuspensionView.ResumeWith] report completion by yielding no further
// suspension frontier (a nil *[kont.Suspension], or a [SuspensionView] whose
// [SuspensionView.Suspension] field is nil). The completed payload returned
// in that case is the zero value of A. Suspended steps may also return the
// zero value of A, so callers must use the suspension/frontier result—not A
// itself, and for [SuspensionView] specifically its
// [SuspensionView.Suspension] field—to detect completion.
// Computations whose result type A is a nilable type (pointer, interface, map, slice,
// channel, or function) therefore cannot use nil as a meaningful completed
// value; wrap nil in an explicit sum or witness type when that distinction
// matters. Carrier context is unaffected: [SuspensionView.Ask] still returns
// the ambient context after completion.
//
// Bridge helpers:
//
//   - [Step], [StepExpr], [Reify], [Reflect], [ReifyReq], and [ReflectReq] remain available as convenience wrappers around kont
//
// Laws:
//
//   - [Duplicate](v).Extract() == v
//   - [Extend](v, func(w View[C, A]) A { return w.Extract() }) == v
//   - [Compose](g, f)(v) == g([Extend](v, f))
//   - [Compose]([ExtractCmd], f) == f and [Compose](g, [ExtractCmd]) == g
//   - NeedExpr(ctx, [ReifyReq](req)) == [Need](ctx, req), when req != nil
//   - [Need](ctx, [ReflectReq](expr)) == [NeedExpr](ctx, expr)
//   - [ExprPullback] distributes over [ExprNot], [ExprAll], and [ExprAny]
package cove
