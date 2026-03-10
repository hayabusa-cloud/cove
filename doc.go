// ©Hayabusa Cloud Co., Ltd. 2026. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// Package cove carries ambient context across [kont] suspension boundaries.
//
// When an external runtime steps through [kont] suspensions one at a time,
// each suspended operation may need state that lives outside the operation
// itself: dispatch budget, ring capabilities, protocol phase, buffer-group
// validity. cove pairs that context with suspensions and values so it travels
// through async boundaries explicitly, without side maps or hidden globals.
//
// The package is policy-free. It carries and checks context but does not
// schedule, retry, or dispatch. Scheduling belongs to takt; kernel mechanics
// to uring; outcome classification to iox.
//
// Carriers:
//
//   - [View[C, A]] — a value paired with ambient context
//   - [SuspensionView[C, A]] — a [kont.Suspension] paired with ambient context
//   - [Req[C]] / [ReqExpr[C]] — context predicates in closure and data forms
//   - [Rule[C]] / [RuleExpr[C]] with [Report] — named predicates for diagnostics
//   - [Checked[C, A]] / [CheckedExpr[C, A]] — requirement-gated values
//   - [Guarded[C, A]] / [GuardedExpr[C, A]] — rule-gated values
//
// Stepping:
//
//   - [StepWith] / [StepExprWith] evaluate a kont computation and pair the first suspension with context
//   - [SuspensionView.Resume] advances a suspension, preserving context
//   - [SuspensionView.ResumeWith] advances a suspension and evolves context
//   - [CheckSuspension] / [CheckSuspensionExpr] gate contextualization on a requirement
//
// Laws:
//
//   - [Duplicate](v).Extract() == v
//   - [Extend](v, func(w View[C, A]) A { return w.Extract() }) == v
//   - NeedExpr(ctx, [ReifyReq](req)) == [Need](ctx, req), when req != nil
//   - [Need](ctx, [ReflectReq](expr)) == [NeedExpr](ctx, expr)
//   - [ExprPullback] distributes over [ExprNot], [ExprAll], and [ExprAny]
package cove
