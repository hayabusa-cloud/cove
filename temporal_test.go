// ©Hayabusa Cloud Co., Ltd. 2026. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package cove_test

import (
	"slices"
	"testing"

	"code.hybscloud.com/cove"
	"code.hybscloud.com/kont"
	"code.hybscloud.com/tempo"
)

func TestTemporalRefusalAndRevocation(t *testing.T) {
	ready := func(obs tempo.Pair[bool, temporalEvidence]) bool { return obs.Left }
	completed := func(obs tempo.Pair[bool, temporalEvidence]) bool { return obs.Right.done }
	req := cove.All(ready, completed)
	expr := cove.ExprAll(cove.ExprAtom(ready), cove.ExprAtom(completed))
	for _, expression := range []bool{false, true} {
		for _, events := range []struct{ first, second, last temporalEvent }{
			{temporalReady, temporalComplete, temporalOther},
			{temporalComplete, temporalReady, temporalOther},
			{temporalReady, temporalComplete, temporalExpire},
			{temporalComplete, temporalReady, temporalCancel},
		} {
			calls := 0
			computation := kont.Bind(kont.Perform(temporalValue{input: 41}), func(v int) kont.Eff[int] {
				calls++
				return kont.Pure(v)
			})
			state := tempo.Both(temporalArmed, temporalEvidence{})
			_, susp := cove.Step(computation)
			check := func() (cove.SuspensionView[tempo.Pair[bool, temporalEvidence], int], bool) {
				if expression {
					return cove.CheckSuspensionExpr(state.Observe(), susp, expr)
				}
				return cove.CheckSuspension(state.Observe(), susp, req)
			}
			refused, ok := check()
			if ok || refused.Suspension != nil || calls != 0 || susp.Op().(temporalValue).input != 41 {
				t.Fatal("refusal consumed or advanced the owner")
			}
			state = state.Step(events.first)
			if refused, ok := check(); ok || refused.Suspension != nil || calls != 0 {
				t.Fatalf("one event admitted work: first=%v expression=%t", events.first, expression)
			}
			state = state.Step(events.second)
			admitted, ok := check()
			if !ok || admitted.Suspension != susp {
				t.Fatalf("events=%v expression=%t: matching facts must admit the original suspension", events, expression)
			}
			state = state.Step(events.last)
			checked, ok := check()
			if ok != (events.last == temporalOther) {
				t.Fatalf("events=%v expression=%t: allowed after final event=%t", events, expression, ok)
			}
			if !ok {
				susp.Discard()
				if calls != 0 {
					t.Fatal("discard ran the continuation")
				}
				continue
			}
			op := checked.Op().(temporalValue)
			value, next := checked.Resume(op.input + 1)
			if value != 42 || next.Suspension != nil || calls != 1 {
				t.Fatal("admitted continuation did not complete once")
			}
		}
	}
}

func TestTemporalCorrelationAndEvidence(t *testing.T) {
	state := tempo.Both(temporalArmed, temporalEvidence{})
	const generation = 7
	accept := func(id uint64, e temporalEvent) {
		if id != generation {
			return
		}
		state = state.Step(e)
	}
	accept(6, temporalReady)
	if state.Observe().Left {
		t.Fatal("stale fact changed this incarnation")
	}
	accept(7, temporalReady)
	accept(7, temporalReturned)
	if obs := state.Observe(); !obs.Left || obs.Right.done || !obs.Right.returned {
		t.Fatal("non-data evidence was conflated with eligibility or completion")
	}
	accept(7, temporalCancel)
	accept(7, temporalComplete)
	if obs := state.Observe(); obs.Left || !obs.Right.done || !obs.Right.returned {
		t.Fatal("eligibility loss erased completion evidence")
	}
	if temporalIdle.Observe() != temporalArmed.Observe() ||
		temporalIdle.Step(temporalReady).Observe() == temporalArmed.Step(temporalReady).Observe() {
		t.Fatal("current observations lost the residual distinction")
	}
}

func TestTemporalMapperOrder(t *testing.T) {
	var order []string
	program := kont.Bind(kont.Perform(temporalValue{input: 1}), func(v int) kont.Eff[int] {
		order = append(order, "first")
		return kont.Bind(kont.Perform(temporalValue{input: v}), func(w int) kont.Eff[int] {
			order = append(order, "second")
			return kont.Pure(w)
		})
	})
	_, owner := cove.StepWith(0, program)
	mapper := func(c int) int { order = append(order, "map"); return c + 1 }
	_, next := owner.ResumeWith(2, mapper)
	if next.Suspension == nil || next.Ctx != 1 {
		t.Fatal("successor missing")
	}
	result, done := next.ResumeWith(3, mapper)
	if result != 3 || done.Suspension != nil || done.Ctx != 2 ||
		!slices.Equal(order, []string{"first", "map", "second", "map"}) {
		t.Fatalf("post-resume order: %v", order)
	}
}

func TestTemporalCompletedValues(t *testing.T) {
	for _, value := range []*int{nil, new(42)} {
		result, done := cove.StepWith(7, kont.Pure(value))
		if done.Suspension != nil || result != value || done.Ctx != 7 {
			t.Fatal("completed pointer result was changed")
		}
	}
	result, done := cove.StepWith(7, kont.Pure(0))
	if done.Suspension != nil || result != 0 {
		t.Fatal("zero result must still complete")
	}
}

func TestTemporalNilInterfaceResults(t *testing.T) {
	for _, value := range []any{nil, (*int)(nil)} {
		for _, expression := range []bool{false, true} {
			for _, suspended := range []bool{false, true} {
				program := kont.Pure(value)
				if suspended {
					program = kont.Bind(kont.Perform(ping{}), func(int) kont.Eff[any] {
						return kont.Pure(value)
					})
				}
				var result any
				var view cove.SuspensionView[int, any]
				if expression {
					result, view = cove.StepExprWith(7, cove.Reify(program))
				} else {
					result, view = cove.StepWith(7, program)
				}
				if suspended {
					if view.Suspension == nil {
						t.Fatal("missing suspension before nil completion")
					}
					result, view = view.Resume(0)
				}
				if view.Suspension != nil || view.Ctx != 7 || result != value {
					t.Fatalf("expression=%v suspended=%v: got %#v, want %#v", expression, suspended, result, value)
				}
			}
		}
	}
}

func TestTemporalForcingBoundary(t *testing.T) {
	eligibility := cove.Force(cove.TotalWorlds[temporalLife](), temporalLife.Observe)
	if eligibility.MonotoneAt(temporalLive, temporalLive.Step(temporalExpire)) {
		t.Fatal("revocable eligibility treated as persistent forcing")
	}
	leq := cove.Preorder[temporalEvidence](func(a, b temporalEvidence) bool {
		return (!a.done || b.done) && (!a.returned || b.returned)
	})
	settled := cove.Force(leq, func(s temporalEvidence) bool { return s.done })
	for _, s := range []temporalEvidence{{}, {done: true}, {returned: true}, {true, true}} {
		for e := temporalArm; e <= temporalOther; e++ {
			if !settled.MonotoneAt(s, s.Step(e)) {
				t.Fatal("accumulated fixed-incarnation evidence lost persistence")
			}
		}
	}
}
