// ©Hayabusa Cloud Co., Ltd. 2026. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package cove

import "code.hybscloud.com/kont"

// Operation is the kont boundary payload shape.
type Operation = kont.Operation

// Resumed is the kont boundary resumption value shape.
type Resumed = kont.Resumed

// SuspensionView pairs a kont suspension with ambient context.
// The exposed Op/Resume shape keeps the carrier structurally joinable with
// `takt` without giving the runner ownership of the carried context.
// A nil Suspension means the computation has completed; Op, Resume,
// ResumeWith, and Discard panic if called after completion.
type SuspensionView[C Ambient, A Focus] struct {
	Ctx        C
	Suspension *kont.Suspension[A]
}

// ObserveSuspension contextualizes a kont suspension.
func ObserveSuspension[C Ambient, A Focus](ctx C, susp *kont.Suspension[A]) SuspensionView[C, A] {
	return SuspensionView[C, A]{Ctx: ctx, Suspension: susp}
}

// CheckSuspension contextualizes a kont suspension when req holds.
func CheckSuspension[C Ambient, A Focus](ctx C, susp *kont.Suspension[A], req Req[C]) (SuspensionView[C, A], bool) {
	if !Need(ctx, req) {
		var zero SuspensionView[C, A]
		return zero, false
	}
	return ObserveSuspension(ctx, susp), true
}

// CheckSuspensionExpr contextualizes a suspension when an Expr-world requirement holds.
func CheckSuspensionExpr[C Ambient, A Focus](ctx C, susp *kont.Suspension[A], req ReqExpr[C]) (SuspensionView[C, A], bool) {
	if !NeedExpr(ctx, req) {
		var zero SuspensionView[C, A]
		return zero, false
	}
	return ObserveSuspension(ctx, susp), true
}

// Extract returns the pending suspension.
func (v SuspensionView[C, A]) Extract() *kont.Suspension[A] { return v.Suspension }

// Ask returns the ambient context.
func (v SuspensionView[C, A]) Ask() C { return v.Ctx }

// MapContextSuspension transports a suspension view across a context mapping
// while preserving the observed suspension frontier.
// It is the [SuspensionView] counterpart of [MapContext].
func MapContextSuspension[C, D Ambient, A Focus](v SuspensionView[C, A], f func(C) D) SuspensionView[D, A] {
	return SuspensionView[D, A]{Ctx: f(v.Ctx), Suspension: v.Suspension}
}

// WithContextSuspension substitutes the ambient context while preserving the
// observed suspension frontier.
// It is the [SuspensionView] counterpart of [WithContext].
func WithContextSuspension[C Ambient, A Focus](v SuspensionView[C, A], ctx C) SuspensionView[C, A] {
	return SuspensionView[C, A]{Ctx: ctx, Suspension: v.Suspension}
}

func (v SuspensionView[C, A]) pending() *kont.Suspension[A] {
	if v.Suspension == nil {
		panic("cove: suspension view completed")
	}
	return v.Suspension
}

// Op returns the suspended operation, and it panics if the computation has completed.
func (v SuspensionView[C, A]) Op() Operation { return v.pending().Op() }

// Resume advances the suspension and keeps the current context, and it panics if the computation has completed.
func (v SuspensionView[C, A]) Resume(value Resumed) (A, SuspensionView[C, A]) {
	return v.ResumeWith(value, nil)
}

// ResumeWith advances the suspension after optionally mapping the context for the next step, and it panics if the computation has completed.
// A nil f preserves the current context, making ResumeWith a total generalization of Resume.
func (v SuspensionView[C, A]) ResumeWith(value Resumed, f func(C) C) (A, SuspensionView[C, A]) {
	result, next := v.pending().Resume(value)
	if next == nil {
		if f == nil {
			return result, SuspensionView[C, A]{Ctx: v.Ctx}
		}
		return result, SuspensionView[C, A]{Ctx: f(v.Ctx)}
	}
	if f == nil {
		return result, SuspensionView[C, A]{Ctx: v.Ctx, Suspension: next}
	}
	return result, SuspensionView[C, A]{Ctx: f(v.Ctx), Suspension: next}
}

// Discard consumes the suspension without resuming it, and it panics if the computation has completed.
func (v SuspensionView[C, A]) Discard() {
	v.pending().Discard()
}

// Step re-exports [kont.Step].
// Prefer [StepWith] when the suspension should stay paired with explicit context.
func Step[A Focus](m kont.Cont[Resumed, A]) (A, *kont.Suspension[A]) {
	return kont.Step(m)
}

// StepExpr re-exports [kont.StepExpr].
// Prefer [StepExprWith] when the suspension should stay paired with explicit context.
func StepExpr[A Focus](m kont.Expr[A]) (A, *kont.Suspension[A]) {
	return kont.StepExpr(m)
}

// StepWith runs a Cont-world computation and pairs the first suspension with ctx.
// This is the primary cove stepping entry point for contextual execution.
func StepWith[C Ambient, A Focus](ctx C, m kont.Cont[Resumed, A]) (A, SuspensionView[C, A]) {
	result, susp := kont.Step(m)
	return result, ObserveSuspension(ctx, susp)
}

// StepExprWith runs an Expr-world computation and pairs the first suspension with ctx.
// This is the primary cove stepping entry point when the caller already works in Expr form.
func StepExprWith[C Ambient, A Focus](ctx C, m kont.Expr[A]) (A, SuspensionView[C, A]) {
	result, susp := kont.StepExpr(m)
	return result, ObserveSuspension(ctx, susp)
}
