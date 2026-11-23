// ©Hayabusa Cloud Co., Ltd. 2026. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package cove

import "code.hybscloud.com/kont"

// Operation is the payload type carried by a kont suspension boundary.
type Operation = kont.Operation

// Resumed is the value type used to resume a kont suspension boundary.
type Resumed = kont.Resumed

// SuspensionView pairs ambient context with a pointer to a [kont.Suspension].
// Copies of the view share that pointer and its one-shot continuation.
// Its Op and Resume methods satisfy [code.hybscloud.com/takt.SuspensionLike].
// A completed step returns a view with a nil Suspension. Op, Resume,
// ResumeWith, and Discard panic on a nil Suspension.
type SuspensionView[C Ambient, A Focus] struct {
	Ctx        C
	Suspension *kont.Suspension[A]
}

// ObserveSuspension returns a suspension view for ctx and susp.
func ObserveSuspension[C Ambient, A Focus](ctx C, susp *kont.Suspension[A]) SuspensionView[C, A] {
	return SuspensionView[C, A]{Ctx: ctx, Suspension: susp}
}

// CheckSuspension returns ctx and susp as a view when req holds, even if susp is nil.
// Refusal returns a zero view and leaves any pending suspension with the caller.
// The caller supplies current context and protects the checked condition through
// the action it authorizes. This check neither dispatches nor validates a result.
func CheckSuspension[C Ambient, A Focus](ctx C, susp *kont.Suspension[A], req Req[C]) (SuspensionView[C, A], bool) {
	if !Need(ctx, req) {
		var zero SuspensionView[C, A]
		return zero, false
	}
	return ObserveSuspension(ctx, susp), true
}

// CheckSuspensionExpr is [CheckSuspension] for an expression requirement.
// Its verdict depends only on req and ctx, including when susp is nil.
func CheckSuspensionExpr[C Ambient, A Focus](ctx C, susp *kont.Suspension[A], req ReqExpr[C]) (SuspensionView[C, A], bool) {
	if !NeedExpr(ctx, req) {
		var zero SuspensionView[C, A]
		return zero, false
	}
	return ObserveSuspension(ctx, susp), true
}

// Extract returns the underlying suspension.
func (v SuspensionView[C, A]) Extract() *kont.Suspension[A] { return v.Suspension }

// Ask returns the ambient context.
func (v SuspensionView[C, A]) Ask() C { return v.Ctx }

// MapContextSuspension maps the ambient context and preserves the observed
// suspension frontier.
func MapContextSuspension[C, D Ambient, A Focus](v SuspensionView[C, A], f func(C) D) SuspensionView[D, A] {
	return SuspensionView[D, A]{Ctx: f(v.Ctx), Suspension: v.Suspension}
}

// WithContextSuspension replaces the ambient context and preserves the observed
// suspension frontier.
func WithContextSuspension[C Ambient, A Focus](v SuspensionView[C, A], ctx C) SuspensionView[C, A] {
	return SuspensionView[C, A]{Ctx: ctx, Suspension: v.Suspension}
}

func (v SuspensionView[C, A]) pending() *kont.Suspension[A] {
	if v.Suspension == nil {
		panic("cove: suspension view completed")
	}
	return v.Suspension
}

// Op returns the suspended operation.
// It panics after completion.
func (v SuspensionView[C, A]) Op() Operation { return v.pending().Op() }

// Resume advances the suspension and keeps the current context.
// When the resumed computation completes, the returned value follows kont's
// nil-completion convention: a nil completed payload becomes the zero value of A.
// Callers must use the returned [SuspensionView] (which has a nil suspension on
// completion) to detect completion, rather than testing the value of A.
// It panics after completion.
func (v SuspensionView[C, A]) Resume(value Resumed) (A, SuspensionView[C, A]) {
	return v.ResumeWith(value, nil)
}

// ResumeWith advances the suspension and optionally maps the context for the
// next step.
// When the resumed computation completes, the returned value follows kont's
// nil-completion convention: a nil completed payload becomes the zero value of A.
// Callers must use the returned [SuspensionView] (which has a nil suspension on
// completion) to detect completion, rather than testing the value of A.
// It panics after completion.
// A nil mapper keeps the current context, so ResumeWith strictly generalizes
// [Resume].
// The mapper runs after the continuation advances, including on completion.
// Refresh context and check any pre-resume requirement before calling ResumeWith;
// advance only the returned successor. Resumption does not settle external I/O.
func (v SuspensionView[C, A]) ResumeWith(value Resumed, f func(C) C) (A, SuspensionView[C, A]) {
	result, next := v.pending().Resume(value)
	if f != nil {
		v.Ctx = f(v.Ctx)
	}
	v.Suspension = next
	return result, v
}

// Discard consumes the suspension without resuming it.
// It panics after completion.
func (v SuspensionView[C, A]) Discard() {
	v.pending().Discard()
}

// Step re-exports [kont.Step].
// Prefer [StepWith] when the suspension should remain paired with explicit
// context. As with [kont.Step], a nil suspension result denotes completion;
// the returned A follows kont's nil-completion convention (a nil payload
// becomes the zero value of A).
func Step[A Focus](m kont.Cont[Resumed, A]) (A, *kont.Suspension[A]) {
	return kont.Step(m)
}

// StepExpr re-exports [kont.StepExpr].
// Prefer [StepExprWith] when the suspension should remain paired with explicit
// context. As with [kont.StepExpr], a nil suspension result denotes completion;
// the returned A follows kont's nil-completion convention (a nil payload
// becomes the zero value of A).
func StepExpr[A Focus](m kont.Expr[A]) (A, *kont.Suspension[A]) {
	return kont.StepExpr(m)
}

// StepWith runs a Cont computation and pairs its first suspension with ctx.
// It is the primary contextual stepping entry point. Completion is observed
// when the returned [SuspensionView.Suspension] is nil; in that case, the
// result A is the completed value, which may itself be nil.
func StepWith[C Ambient, A Focus](ctx C, m kont.Cont[Resumed, A]) (A, SuspensionView[C, A]) {
	result, susp := kont.Step(m)
	return result, ObserveSuspension(ctx, susp)
}

// StepExprWith runs an Expr computation and pairs its first suspension with ctx.
// It is the primary contextual stepping entry point for callers already in Expr
// form. Completion is observed when the returned [SuspensionView.Suspension]
// is nil; in that case, the result A is the completed value, which may itself
// be nil.
func StepExprWith[C Ambient, A Focus](ctx C, m kont.Expr[A]) (A, SuspensionView[C, A]) {
	result, susp := kont.StepExpr(m)
	return result, ObserveSuspension(ctx, susp)
}
