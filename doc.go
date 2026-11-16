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
// Context is passed explicitly. View operations copy their fields; they do not
// clone referenced storage or grant additional suspension ownership. Callers
// keep shared context and requirement operands stable during evaluation and
// provide any needed concurrency control.
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
//   - [View], a focused value paired with ambient context
//   - [Cmd], a contextual command over [View]
//   - [SuspensionView], a [kont.Suspension] paired with ambient context
//   - [World], [StepIndex], [Preorder], [Transition], [Forcing],
//     [ForcingExpr], [Indexed], and [Relation], step-indexed Kripke evidence over ambient worlds
//   - [IndexedView] and [IndexedSuspensionView], contextual
//     observations paired with explicit finite step credit
//   - [Req] / [ReqExpr], context predicates in closure and data forms
//   - [Rule] / [RuleExpr] with [Report], named predicates for diagnostics
//   - [Checked] / [CheckedExpr], requirement-gated values
//   - [Guarded] / [GuardedExpr], rule-gated values
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
//   - [Preorder] names world extension, w <= w'
//   - [DiscreteWorlds] and [TotalWorlds] provide equality-only and total preorders
//   - [Extends] is the package-level world-extension helper
//   - [Preorder.Extends], [Preorder.ReflexiveAt], and [Preorder.TransitiveAt] expose concrete preorder law checks
//   - [Transition], [CheckTransition], [CheckForcingTransition], and [CheckInvariantTransition] check concrete world edges
//   - [Force] / [ForceExpr] construct [Forcing] and [ForcingExpr] evidence with monotonicity checks
//   - [Relation] is indexed by world, [StepIndex], and value
//   - [Later] implements the guarded later modality: index zero is trivial; index n+1 checks at n
//   - [CheckRelation], [WeakenIndexedView], and [WeakenIndexedSuspension] check indexed evidence or weaken finite credit
//   - [CheckCompletedRelation] checks the final finite-trace relation at a completed indexed frontier
//
// [Step] and [StepExpr] report completion with a nil *[kont.Suspension].
// [StepWith], [StepExprWith], [SuspensionView.Resume], and
// [SuspensionView.ResumeWith] report completion with a nil
// [SuspensionView.Suspension] field. Use that field to detect completion;
// pending steps may also return the zero value of A. The completed value follows
// kont's classifier: a nil payload becomes the zero value of A, and a typed nil
// inside an interface retains its dynamic type. Nil is a valid completed value
// for a nilable result type. [SuspensionView.Ask] preserves the carried context
// after completion.
//
// A refused requirement check leaves the original suspension unconsumed. Keep
// that handle, refresh context from its owner, and check before the action being
// authorized. The owner must protect the interval between check and action.
// ResumeWith maps context after advancement; its mapper is not a pre-resume
// guard. ResumeTo checks a supplied world edge without acquiring fresh context.
// Revocable eligibility is not persistent forcing, and StepIndex is not a clock.
// Discard consumes continuation control; external requests and resource loans
// retain their independent completion and release obligations.
//
// Bridge helpers:
//
//   - [Step], [StepExpr], [Reify], and [Reflect] wrap kont's computation forms without binding context
//   - [ReifyReq] and [ReflectReq] bridge cove's closure and expression requirements
//
// Laws below assume pure, total callbacks and stable context. Equality is
// observational; it does not require values or functions to be Go-comparable.
// ExprPullback may apply its context projection to each visited atom. Bind
// composite requirements before the evaluation path when their construction
// cost should be excluded from that path.
//
//   - [Duplicate](v).Extract() == v
//   - [Extend](v, func(w View[C, A]) A { return w.Extract() }) == v
//   - [Compose](g, f)(v) == g([Extend](v, f))
//   - [Compose]([ExtractCmd], f) == f and [Compose](g, [ExtractCmd]) == g
//   - NeedExpr(ctx, [ReifyReq](req)) == [Need](ctx, req), when req != nil
//   - [Need](ctx, [ReflectReq](expr)) == [NeedExpr](ctx, expr)
//   - [ExprPullback] distributes over [ExprNot], [ExprAll], and [ExprAny]
package cove
