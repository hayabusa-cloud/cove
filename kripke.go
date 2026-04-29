// ©Hayabusa Cloud Co., Ltd. 2026. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package cove

import "code.hybscloud.com/kont"

// StepIndex is the finite approximation level used by step-indexed
// contextual interpretations.
type StepIndex = kont.StepIndex

// Preorder is the Kripke world-extension relation on ambient contexts.
//
// A call leq(w, wNext) means w <= wNext: wNext is an admissible future
// refinement of w.
type Preorder[C World] func(w, wNext C) bool

func requirePreorder[C World](leq Preorder[C]) {
	if leq == nil {
		panic("cove: nil world preorder")
	}
}

// Extends reports whether wNext is an admissible future refinement of w.
func (leq Preorder[C]) Extends(w, wNext C) bool {
	requirePreorder(leq)
	return leq(w, wNext)
}

// ReflexiveAt checks the preorder reflexivity law at w.
func (leq Preorder[C]) ReflexiveAt(w C) bool {
	return leq.Extends(w, w)
}

// TransitiveAt checks the preorder transitivity law at w0 <= w1 <= w2.
func (leq Preorder[C]) TransitiveAt(w0, w1, w2 C) bool {
	if !leq.Extends(w0, w1) || !leq.Extends(w1, w2) {
		return true
	}
	return leq.Extends(w0, w2)
}

// DiscreteWorlds returns equality as a world preorder for comparable contexts.
func DiscreteWorlds[C comparable]() Preorder[C] {
	return func(w, wNext C) bool { return w == wNext }
}

// TotalWorlds returns the preorder where every world refines every other world.
func TotalWorlds[C World]() Preorder[C] {
	return func(C, C) bool { return true }
}

// Extends reports whether wNext is an admissible future refinement of w.
func Extends[C World](leq Preorder[C], w, wNext C) bool {
	return leq.Extends(w, wNext)
}

// Transition is a concrete candidate edge between two Kripke worlds.
type Transition[C World] struct {
	Before C
	After  C
}

// CheckTransition reports whether t.After is an admissible successor of
// t.Before under leq.
func CheckTransition[C World](leq Preorder[C], t Transition[C]) bool {
	return leq.Extends(t.Before, t.After)
}

// Forcing is a requirement together with the world preorder under which the
// requirement must be monotone.
type Forcing[C World] struct {
	leq Preorder[C]
	req Req[C]
}

// Force constructs forcing evidence over a world preorder.
//
// The caller supplies the predicate and the preorder; [Forcing.MonotoneAt]
// checks the monotonicity obligation at concrete world pairs.
func Force[C World](leq Preorder[C], req Req[C]) Forcing[C] {
	requirePreorder(leq)
	return Forcing[C]{leq: leq, req: req}
}

// Preorder returns the world-extension relation carried by f.
func (f Forcing[C]) Preorder() Preorder[C] {
	requirePreorder(f.leq)
	return f.leq
}

// Req returns the underlying requirement.
func (f Forcing[C]) Req() Req[C] { return f.req }

// Holds reports whether world satisfies f.
func (f Forcing[C]) Holds(world C) bool { return Need(world, f.req) }

// MonotoneAt checks the forcing monotonicity obligation at w <= wNext.
//
// The law is: if w <= wNext and f holds at w, then f holds at wNext.
func (f Forcing[C]) MonotoneAt(w, wNext C) bool {
	if !Extends(f.leq, w, wNext) || !f.Holds(w) {
		return true
	}
	return f.Holds(wNext)
}

// CheckForcingTransition checks that a concrete transition extends the world
// and preserves f at both endpoints.
func CheckForcingTransition[C World](f Forcing[C], t Transition[C]) bool {
	return CheckTransition(f.Preorder(), t) &&
		f.Holds(t.Before) &&
		f.MonotoneAt(t.Before, t.After) &&
		f.Holds(t.After)
}

// CheckInvariantTransition checks that req holds across a concrete transition
// under leq.
func CheckInvariantTransition[C World](leq Preorder[C], req Req[C], t Transition[C]) bool {
	return CheckForcingTransition(Force(leq, req), t)
}

// ForcingExpr is the ReqExpr counterpart of [Forcing].
type ForcingExpr[C World] struct {
	leq Preorder[C]
	req ReqExpr[C]
}

// ForceExpr constructs expression-form forcing evidence over a world preorder.
func ForceExpr[C World](leq Preorder[C], req ReqExpr[C]) ForcingExpr[C] {
	requirePreorder(leq)
	return ForcingExpr[C]{leq: leq, req: req}
}

// Preorder returns the world-extension relation carried by f.
func (f ForcingExpr[C]) Preorder() Preorder[C] {
	requirePreorder(f.leq)
	return f.leq
}

// Req returns the underlying expression requirement.
func (f ForcingExpr[C]) Req() ReqExpr[C] { return f.req }

// Holds reports whether world satisfies f.
func (f ForcingExpr[C]) Holds(world C) bool { return NeedExpr(world, f.req) }

// MonotoneAt checks expression-form forcing monotonicity at w <= wNext.
func (f ForcingExpr[C]) MonotoneAt(w, wNext C) bool {
	if !Extends(f.leq, w, wNext) || !f.Holds(w) {
		return true
	}
	return f.Holds(wNext)
}

// Indexed is a step-indexed predicate over world, approximation level, and value.
type Indexed[C World, A Focus] func(world C, index StepIndex, value A) bool

// Relation is a step-indexed Kripke logical relation over contextual values.
type Relation[C World, A Focus] struct {
	leq  Preorder[C]
	hold Indexed[C, A]
}

// Relate constructs a step-indexed Kripke relation.
func Relate[C World, A Focus](leq Preorder[C], hold Indexed[C, A]) Relation[C, A] {
	requirePreorder(leq)
	if hold == nil {
		panic("cove: nil indexed relation")
	}
	return Relation[C, A]{leq: leq, hold: hold}
}

// Preorder returns the world-extension relation carried by r.
func (r Relation[C, A]) Preorder() Preorder[C] {
	requirePreorder(r.leq)
	return r.leq
}

// Holds reports whether value belongs to r at world and index.
func (r Relation[C, A]) Holds(world C, index StepIndex, value A) bool {
	if r.hold == nil {
		panic("cove: nil indexed relation")
	}
	return r.hold(world, index, value)
}

// MonotoneAt checks Kripke monotonicity at w <= wNext for a fixed index.
func (r Relation[C, A]) MonotoneAt(w, wNext C, index StepIndex, value A) bool {
	if !Extends(r.leq, w, wNext) || !r.Holds(w, index, value) {
		return true
	}
	return r.Holds(wNext, index, value)
}

// WeakensAt checks step-index weakening for a fixed world.
//
// The law is: if strong allows weak and r holds at strong, then r holds at weak.
func (r Relation[C, A]) WeakensAt(world C, strong, weak StepIndex, value A) bool {
	if !strong.Allows(weak) || !r.Holds(world, strong, value) {
		return true
	}
	return r.Holds(world, weak, value)
}

// Later delays r by one physical step.
//
// At index zero, Later is guarded and holds trivially. At n+1, it checks r at n.
func Later[C World, A Focus](r Relation[C, A]) Relation[C, A] {
	return Relate(r.Preorder(), func(world C, index StepIndex, value A) bool {
		prev, ok := index.Prev()
		if !ok {
			return true
		}
		return r.Holds(world, prev, value)
	})
}

// IndexedView pairs a contextual value with an explicit step index.
type IndexedView[C World, A Focus] struct {
	Index StepIndex
	View  View[C, A]
}

// ObserveIndexed returns an indexed view for ctx, index, and value.
func ObserveIndexed[C World, A Focus](ctx C, index StepIndex, value A) IndexedView[C, A] {
	return IndexedView[C, A]{Index: index, View: Observe(ctx, value)}
}

// Ask returns the ambient context.
func (v IndexedView[C, A]) Ask() C { return v.View.Ask() }

// Extract returns the focused value.
func (v IndexedView[C, A]) Extract() A { return v.View.Extract() }

// CheckRelation reports whether v satisfies r at its carried world and index.
func CheckRelation[C World, A Focus](v IndexedView[C, A], r Relation[C, A]) bool {
	return r.Holds(v.Ask(), v.Index, v.Extract())
}

// WeakenIndexedView returns v observed at target when target is a valid
// weakening of v's current step index.
func WeakenIndexedView[C World, A Focus](v IndexedView[C, A], target StepIndex) (IndexedView[C, A], bool) {
	index, ok := v.Index.Weaken(target)
	if !ok {
		var zero IndexedView[C, A]
		return zero, false
	}
	v.Index = index
	return v, true
}

// IndexedSuspensionView pairs a contextual suspension with explicit step fuel.
type IndexedSuspensionView[C World, A Focus] struct {
	Index StepIndex
	View  SuspensionView[C, A]
}

// ObserveIndexedSuspension returns an indexed suspension view.
func ObserveIndexedSuspension[C World, A Focus](ctx C, index StepIndex, susp *kont.Suspension[A]) IndexedSuspensionView[C, A] {
	return IndexedSuspensionView[C, A]{Index: index, View: ObserveSuspension(ctx, susp)}
}

// Ask returns the ambient context.
func (v IndexedSuspensionView[C, A]) Ask() C { return v.View.Ask() }

// Extract returns the underlying suspension.
func (v IndexedSuspensionView[C, A]) Extract() *kont.Suspension[A] {
	return v.View.Extract()
}

// Completed reports whether the indexed suspension frontier has completed.
func (v IndexedSuspensionView[C, A]) Completed() bool { return v.Extract() == nil }

// Op returns the suspended operation.
func (v IndexedSuspensionView[C, A]) Op() Operation { return v.View.Op() }

// Resume advances the suspension and consumes one step of fuel. It panics if
// the suspension has completed or if the step index is zero.
func (v IndexedSuspensionView[C, A]) Resume(value Resumed) (A, IndexedSuspensionView[C, A]) {
	return v.ResumeWith(value, nil)
}

// ResumeWith advances the suspension, consumes one step of fuel, and evolves
// the carried world for the successor observation. It panics if the suspension
// has completed or if the step index is zero.
func (v IndexedSuspensionView[C, A]) ResumeWith(value Resumed, f func(C) C) (A, IndexedSuspensionView[C, A]) {
	_ = v.View.pending()
	nextIndex := v.Index.MustPrev()
	result, next := v.View.ResumeWith(value, f)
	return result, IndexedSuspensionView[C, A]{Index: nextIndex, View: next}
}

// ResumeTo advances the suspension to a specified successor world.
//
// It consumes one step of fuel and requires nextWorld to extend the current
// world under leq. Use ResumeTo when the indexed observation is intended as
// Kripke evidence; use ResumeWith only for policy-free context threading where
// the caller proves monotonicity externally. It panics if the suspension has
// completed, if the step index is zero, or if nextWorld does not extend the
// current world under leq.
func (v IndexedSuspensionView[C, A]) ResumeTo(leq Preorder[C], value Resumed, nextWorld C) (A, IndexedSuspensionView[C, A]) {
	_ = v.View.pending()
	if !leq.Extends(v.Ask(), nextWorld) {
		panic("cove: world does not extend")
	}
	nextIndex := v.Index.MustPrev()
	result, next := v.View.ResumeWith(value, func(C) C { return nextWorld })
	return result, IndexedSuspensionView[C, A]{Index: nextIndex, View: next}
}

// WeakenIndexedSuspension returns v observed at target when target is a valid
// weakening of v's current step index.
func WeakenIndexedSuspension[C World, A Focus](v IndexedSuspensionView[C, A], target StepIndex) (IndexedSuspensionView[C, A], bool) {
	index, ok := v.Index.Weaken(target)
	if !ok {
		var zero IndexedSuspensionView[C, A]
		return zero, false
	}
	v.Index = index
	return v, true
}

// CheckCompletedRelation checks the finite-trace adequacy bridge for a
// completed indexed suspension frontier.
//
// It reports false while the frontier is still suspended. After completion, it
// checks whether value is in r at the frontier's final world and remaining
// step index.
func CheckCompletedRelation[C World, A Focus](v IndexedSuspensionView[C, A], value A, r Relation[C, A]) bool {
	if !v.Completed() {
		return false
	}
	return r.Holds(v.Ask(), v.Index, value)
}

// Discard consumes the underlying suspension without resuming it.
func (v IndexedSuspensionView[C, A]) Discard() { v.View.Discard() }

// StepWithIndex runs a Cont computation and pairs its first suspension with ctx
// and an explicit step index.
func StepWithIndex[C World, A Focus](ctx C, index StepIndex, m kont.Cont[Resumed, A]) (A, IndexedSuspensionView[C, A]) {
	result, susp := kont.Step(m)
	return result, ObserveIndexedSuspension(ctx, index, susp)
}

// StepExprWithIndex runs an Expr computation and pairs its first suspension with
// ctx and an explicit step index.
func StepExprWithIndex[C World, A Focus](ctx C, index StepIndex, m kont.Expr[A]) (A, IndexedSuspensionView[C, A]) {
	result, susp := kont.StepExpr(m)
	return result, ObserveIndexedSuspension(ctx, index, susp)
}
