// ©Hayabusa Cloud Co., Ltd. 2026. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package cove_test

import (
	"testing"

	"code.hybscloud.com/cove"
	"code.hybscloud.com/kont"
)

type kripkeWorld struct {
	Budget int
	Phase  int
}

func kripkeLeq(w, next kripkeWorld) bool {
	return w.Budget <= next.Budget && w.Phase <= next.Phase
}

func TestForcingMonotoneAt(t *testing.T) {
	if !cove.DiscreteWorlds[int]().ReflexiveAt(7) {
		t.Fatal("discrete worlds should be reflexive")
	}
	if cove.DiscreteWorlds[int]().Extends(7, 8) {
		t.Fatal("discrete worlds should not relate distinct values")
	}
	if !cove.TotalWorlds[kripkeWorld]().Extends(kripkeWorld{}, kripkeWorld{Phase: 99}) {
		t.Fatal("total worlds should relate every pair")
	}
	if !cove.Preorder[kripkeWorld](kripkeLeq).ReflexiveAt(kripkeWorld{Budget: 1}) {
		t.Fatal("preorder should be reflexive at checked world")
	}
	if !cove.Preorder[kripkeWorld](kripkeLeq).TransitiveAt(
		kripkeWorld{Budget: 2, Phase: 2},
		kripkeWorld{Budget: 1, Phase: 3},
		kripkeWorld{Budget: 4, Phase: 4},
	) {
		t.Fatal("transitivity check should be vacuous when premises fail")
	}
	if !cove.Preorder[kripkeWorld](kripkeLeq).TransitiveAt(
		kripkeWorld{Budget: 1},
		kripkeWorld{Budget: 2},
		kripkeWorld{Budget: 3},
	) {
		t.Fatal("preorder should be transitive at checked worlds")
	}

	forcing := cove.Force[kripkeWorld](kripkeLeq, func(w kripkeWorld) bool {
		return w.Budget >= 2
	})
	if !forcing.Holds(kripkeWorld{Budget: 2}) {
		t.Fatal("forcing should hold at sufficient budget")
	}
	if forcing.Preorder() == nil || forcing.Req() == nil {
		t.Fatal("forcing should expose its preorder and requirement")
	}
	if !forcing.MonotoneAt(kripkeWorld{Budget: 2}, kripkeWorld{Budget: 3}) {
		t.Fatal("upward-closed forcing should be monotone")
	}
	if !forcing.MonotoneAt(kripkeWorld{Budget: 1}, kripkeWorld{Budget: 2}) {
		t.Fatal("monotonicity check should be vacuous when forcing does not hold")
	}

	nonMonotone := cove.Force[kripkeWorld](kripkeLeq, func(w kripkeWorld) bool {
		return w.Budget == 2
	})
	if nonMonotone.MonotoneAt(kripkeWorld{Budget: 2}, kripkeWorld{Budget: 3}) {
		t.Fatal("equality predicate is not upward closed under budget extension")
	}
}

func TestTransitionHelpers(t *testing.T) {
	leq := cove.Preorder[kripkeWorld](kripkeLeq)
	transition := cove.Transition[kripkeWorld]{
		Before: kripkeWorld{Budget: 2, Phase: 1},
		After:  kripkeWorld{Budget: 3, Phase: 2},
	}
	if !cove.CheckTransition(leq, transition) {
		t.Fatal("transition should follow the world preorder")
	}

	req := func(w kripkeWorld) bool { return w.Budget >= 2 }
	if !cove.CheckInvariantTransition(leq, req, transition) {
		t.Fatal("invariant should hold across the transition")
	}
	forcing := cove.Force(leq, req)
	if !cove.CheckForcingTransition(forcing, transition) {
		t.Fatal("forcing transition should hold")
	}

	bad := cove.Transition[kripkeWorld]{
		Before: kripkeWorld{Budget: 3, Phase: 2},
		After:  kripkeWorld{Budget: 2, Phase: 3},
	}
	if cove.CheckTransition(leq, bad) {
		t.Fatal("transition must reject non-successor worlds")
	}
	if cove.CheckInvariantTransition(leq, req, bad) {
		t.Fatal("invariant transition must reject non-successor worlds")
	}
}

func TestForcingExprMonotoneAt(t *testing.T) {
	expr := cove.ExprAtom(func(w kripkeWorld) bool { return w.Phase >= 1 })
	forcing := cove.ForceExpr(kripkeLeq, expr)
	if !forcing.Holds(kripkeWorld{Phase: 1}) {
		t.Fatal("expression forcing should hold")
	}
	if forcing.Preorder() == nil || !cove.NeedExpr(kripkeWorld{Phase: 1}, forcing.Req()) {
		t.Fatal("expression forcing should expose its preorder and requirement")
	}
	if !forcing.MonotoneAt(kripkeWorld{Phase: 1}, kripkeWorld{Phase: 2}) {
		t.Fatal("expression forcing should be monotone at checked worlds")
	}
	if !forcing.MonotoneAt(kripkeWorld{Phase: 0}, kripkeWorld{Phase: 1}) {
		t.Fatal("expression monotonicity should be vacuous when forcing does not hold")
	}

	eqExpr := cove.ExprAtom(func(w kripkeWorld) bool { return w.Phase == 1 })
	nonMonotone := cove.ForceExpr(kripkeLeq, eqExpr)
	if nonMonotone.MonotoneAt(kripkeWorld{Phase: 1}, kripkeWorld{Phase: 2}) {
		t.Fatal("expression equality predicate is not upward closed under phase extension")
	}
}

func TestRelationWeakeningLaterAndMonotonicity(t *testing.T) {
	relation := cove.Relate[kripkeWorld, int](kripkeLeq, func(w kripkeWorld, n cove.StepIndex, value int) bool {
		return w.Budget >= value && uint64(n) <= uint64(value)
	})
	world := kripkeWorld{Budget: 8}
	if !relation.Holds(world, 3, 5) {
		t.Fatal("relation should hold at index 3")
	}
	if !relation.WeakensAt(world, 3, 1, 5) {
		t.Fatal("relation should weaken from index 3 to 1")
	}
	view := cove.ObserveIndexed(world, cove.StepIndex(3), 5)
	weakened, ok := cove.WeakenIndexedView(view, 1)
	if !ok || weakened.Index != 1 || !cove.CheckRelation(weakened, relation) {
		t.Fatalf("indexed view weakening failed: index=%d ok=%v", weakened.Index, ok)
	}
	if _, ok := cove.WeakenIndexedView(view, 4); ok {
		t.Fatal("indexed view must not weaken to a larger index")
	}
	if !relation.MonotoneAt(world, kripkeWorld{Budget: 9}, 3, 5) {
		t.Fatal("relation should be monotone across larger worlds")
	}

	later := cove.Later(relation)
	if !later.Holds(world, 0, 1000) {
		t.Fatal("later relation is guarded at zero")
	}
	if !later.Holds(world, 4, 5) {
		t.Fatal("later relation at n+1 should check the base relation at n")
	}
}

func TestLaterRelationComposes(t *testing.T) {
	relation := cove.Relate[kripkeWorld, int](kripkeLeq, func(w kripkeWorld, n cove.StepIndex, value int) bool {
		return w.Budget >= value && uint64(n) <= uint64(value)
	})
	world := kripkeWorld{Budget: 8}
	laterLater := cove.Later(cove.Later(relation))

	if !laterLater.Holds(world, 0, 0) {
		t.Fatal("double later should remain guarded at zero")
	}
	if !laterLater.Holds(world, 1, 0) {
		t.Fatal("double later should remain guarded after one delay")
	}
	if !laterLater.Holds(world, 2, 0) {
		t.Fatal("double later at n+2 should check base relation at n")
	}
	if laterLater.Holds(world, 3, 0) {
		t.Fatal("double later should expose base relation failure after two delays")
	}
	if !laterLater.WeakensAt(world, 3, 2, 0) {
		t.Fatal("double later weakening should compose with base relation")
	}
}

func TestLaterRelationUsesWorldDependentBase(t *testing.T) {
	relation := cove.Relate[kripkeWorld, int](kripkeLeq, func(w kripkeWorld, n cove.StepIndex, value int) bool {
		return w.Budget >= int(n)+value
	})
	later := cove.Later(relation)

	if !later.Holds(kripkeWorld{Budget: 6}, 3, 4) {
		t.Fatal("later should check world-dependent relation at the predecessor index")
	}
	if later.Holds(kripkeWorld{Budget: 5}, 3, 4) {
		t.Fatal("later should expose world-dependent relation failure after one delay")
	}
	if !later.MonotoneAt(kripkeWorld{Budget: 6}, kripkeWorld{Budget: 7}, 3, 4) {
		t.Fatal("later should preserve base relation monotonicity across larger worlds")
	}
}

func TestRelationMonotoneAtPartialOrderIncomparable(t *testing.T) {
	relation := cove.Relate[kripkeWorld, int](kripkeLeq, func(w kripkeWorld, _ cove.StepIndex, value int) bool {
		return w.Budget >= value
	})

	w := kripkeWorld{Budget: 2, Phase: 2}
	incomparable := kripkeWorld{Budget: 3, Phase: 1}
	if !relation.MonotoneAt(w, incomparable, 0, 2) {
		t.Fatal("monotonicity check should be vacuous for incomparable worlds")
	}

	comparable := kripkeWorld{Budget: 3, Phase: 3}
	if !relation.MonotoneAt(w, comparable, 0, 2) {
		t.Fatal("monotonicity check should hold for comparable larger worlds")
	}
}

func TestRelationLawFailuresAreObservable(t *testing.T) {
	world := kripkeWorld{}
	eqIndex := cove.Relate[kripkeWorld, int](kripkeLeq, func(_ kripkeWorld, n cove.StepIndex, value int) bool {
		return uint64(n) == uint64(value)
	})
	if eqIndex.WeakensAt(world, 5, 3, 5) {
		t.Fatal("non-downward-closed relation must fail weakening check")
	}
}

func TestIndexedSuspensionViewConsumesStepIndex(t *testing.T) {
	cont := kont.Bind(kont.Perform(ping{}), func(v int) kont.Eff[int] {
		return kont.Bind(kont.Perform(ping{}), func(w int) kont.Eff[int] {
			return kont.Pure(v + w)
		})
	})

	_, step := cove.StepWithIndex(kripkeWorld{Budget: 2}, 2, cont)
	if step.Index != 2 || step.Extract() == nil {
		t.Fatalf("unexpected initial indexed suspension: %#v", step)
	}
	_, step = step.ResumeWith(10, func(w kripkeWorld) kripkeWorld {
		w.Budget--
		return w
	})
	if step.Index != 1 || step.Ask().Budget != 1 || step.Extract() == nil {
		t.Fatalf("unexpected second indexed suspension: index=%d world=%#v susp=%v", step.Index, step.Ask(), step.Extract())
	}
	result, done := step.ResumeWith(20, func(w kripkeWorld) kripkeWorld {
		w.Budget--
		return w
	})
	if done.Index != 0 || done.Ask().Budget != 0 || done.Extract() != nil {
		t.Fatalf("unexpected completed indexed suspension: index=%d world=%#v susp=%v", done.Index, done.Ask(), done.Extract())
	}
	if result != 30 {
		t.Fatalf("unexpected result: got %d want 30", result)
	}
	if !done.Completed() {
		t.Fatal("completed indexed suspension should report completion")
	}
	relation := cove.Relate[kripkeWorld, int](cove.TotalWorlds[kripkeWorld](), func(w kripkeWorld, n cove.StepIndex, value int) bool {
		return w.Budget == 0 && n == 0 && value == 30
	})
	if !cove.CheckCompletedRelation(done, result, relation) {
		t.Fatal("completed finite trace should satisfy the final relation")
	}
}

func TestCheckCompletedRelationUsesProvidedValue(t *testing.T) {
	cont := kont.Bind(kont.Perform(ping{}), func(v int) kont.Eff[int] {
		return kont.Pure(v + 1)
	})

	_, step := cove.StepWithIndex(kripkeWorld{}, 1, cont)
	result, done := step.Resume(4)
	if result != 5 || !done.Completed() {
		t.Fatalf("unexpected completion: result=%d done=%v", result, done.Completed())
	}
	relation := cove.Relate[kripkeWorld, int](cove.TotalWorlds[kripkeWorld](), func(_ kripkeWorld, _ cove.StepIndex, value int) bool {
		return value == 99
	})
	if cove.CheckCompletedRelation(done, result, relation) {
		t.Fatal("actual result should not satisfy relation that expects a distinct witness")
	}
	if !cove.CheckCompletedRelation(done, 99, relation) {
		t.Fatal("completed relation should check the explicitly supplied witness")
	}
}

func TestCheckCompletedRelationWhileSuspended(t *testing.T) {
	_, step := cove.StepExprWithIndex(kripkeWorld{}, 1, kont.ExprPerform(ping{}))
	relation := cove.Relate[kripkeWorld, int](cove.TotalWorlds[kripkeWorld](), func(kripkeWorld, cove.StepIndex, int) bool {
		return true
	})
	if cove.CheckCompletedRelation(step, 0, relation) {
		t.Fatal("suspended frontier must not satisfy completed relation")
	}
	step.Discard()
}

func TestIndexedSuspensionViewOp(t *testing.T) {
	_, step := cove.StepExprWithIndex(kripkeWorld{}, 1, kont.ExprPerform(ping{}))
	if _, ok := step.Op().(ping); !ok {
		t.Fatalf("expected ping operation, got %T", step.Op())
	}
	step.Discard()
}

func TestIndexedSuspensionViewRejectsZeroFuelResume(t *testing.T) {
	_, step := cove.StepExprWithIndex(kripkeWorld{}, 0, kont.ExprPerform(ping{}))
	if step.Extract() == nil {
		t.Fatal("expected suspension")
	}
	expectPanicMessage(t, "kont: step index exhausted", func() {
		step.Resume(1)
	})
	step.Discard()
}

func TestIndexedSuspensionViewResumeToChecksWorldExtension(t *testing.T) {
	leq := cove.Preorder[kripkeWorld](kripkeLeq)
	_, step := cove.StepExprWithIndex(kripkeWorld{Budget: 1}, 1, kont.ExprPerform(ping{}))
	expectPanicMessage(t, "cove: world does not extend", func() {
		step.ResumeTo(leq, 1, kripkeWorld{Budget: 0})
	})

	result, done := step.ResumeTo(leq, 7, kripkeWorld{Budget: 2})
	if result != 7 || done.Index != 0 || done.Ask().Budget != 2 || !done.Completed() {
		t.Fatalf("unexpected monotone resume: result=%d index=%d world=%#v done=%v", result, done.Index, done.Ask(), done.Completed())
	}
}

func TestIndexedSuspensionViewWeakening(t *testing.T) {
	_, step := cove.StepExprWithIndex(kripkeWorld{}, 3, kont.ExprPerform(ping{}))
	weakened, ok := cove.WeakenIndexedSuspension(step, 1)
	if !ok || weakened.Index != 1 || weakened.Extract() == nil {
		t.Fatalf("indexed suspension weakening failed: index=%d ok=%v", weakened.Index, ok)
	}
	if _, ok := cove.WeakenIndexedSuspension(step, 4); ok {
		t.Fatal("indexed suspension must not weaken to a larger index")
	}
	step.Discard()
}

func TestIndexedSuspensionViewCompletedResumePanicsBeforeFuel(t *testing.T) {
	done := cove.ObserveIndexedSuspension[kripkeWorld, int](kripkeWorld{}, 0, nil)
	expectPanicMessage(t, "cove: suspension view completed", func() {
		done.Resume(1)
	})
	expectPanicMessage(t, "cove: suspension view completed", func() {
		done.ResumeTo(cove.TotalWorlds[kripkeWorld](), 1, kripkeWorld{})
	})
}

func TestKripkeConstructorsRejectNilEvidence(t *testing.T) {
	expectPanicMessage(t, "cove: nil world preorder", func() {
		_ = cove.Force[int](nil, nil)
	})
	expectPanicMessage(t, "cove: nil indexed relation", func() {
		var relation cove.Relation[int, int]
		_ = relation.Holds(0, 0, 0)
	})
	expectPanicMessage(t, "cove: nil indexed relation", func() {
		_ = cove.Relate[int, int](cove.DiscreteWorlds[int](), nil)
	})
	expectPanicMessage(t, "cove: nil world preorder", func() {
		_, step := cove.StepExprWithIndex(kripkeWorld{}, 1, kont.ExprPerform(ping{}))
		defer step.Discard()
		step.ResumeTo(nil, 1, kripkeWorld{})
	})
}
