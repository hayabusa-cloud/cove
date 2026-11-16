// ©Hayabusa Cloud Co., Ltd. 2026. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package cove_test

import (
	"slices"
	"testing"

	"code.hybscloud.com/cove"
	"code.hybscloud.com/kont"
)

func TestTemporalRefusalAndRevocation(t *testing.T) {
	for _, expression := range []bool{false, true} {
		for _, terminal := range []temporalEvent{temporalReady, temporalExpire, temporalCancel} {
			calls := 0
			computation := kont.Bind(kont.Perform(temporalValue{input: 41}), func(v int) kont.Eff[int] {
				calls++
				return kont.Pure(v)
			})
			life := temporalArmed
			_, owner := cove.StepWith(temporalContext{generation: 7}, computation)
			check := func() (cove.SuspensionView[temporalContext, int], bool) {
				if expression {
					return cove.CheckSuspensionExpr(owner.Ctx, owner.Suspension, cove.ExprAtom(temporalAllowed))
				}
				return cove.CheckSuspension(owner.Ctx, owner.Suspension, temporalAllowed)
			}
			refused, ok := check()
			if ok || refused.Suspension != nil || calls != 0 || owner.Op().(temporalValue).input != 41 {
				t.Fatal("refusal consumed or advanced the owner")
			}
			life = life.Step(temporalReady)
			cached := life.Observe()
			life = life.Step(terminal)
			owner = cove.WithContextSuspension(owner, temporalContext{life.Observe(), 7})
			checked, ok := check()
			if !cached || ok != (terminal == temporalReady) {
				t.Fatal("fresh eligibility did not account for revocation")
			}
			if !ok {
				owner.Discard()
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
	life, evidence := temporalArmed, temporalEvidence{}
	generation := uint64(7)
	accept := func(id uint64, e temporalEvent) {
		if id != generation {
			return
		}
		life, evidence = life.Step(e), evidence.Step(e)
	}
	accept(6, temporalReady)
	if life.Observe() {
		t.Fatal("stale fact changed this incarnation")
	}
	accept(7, temporalReady)
	accept(7, temporalReturned)
	if !life.Observe() || evidence.done || !evidence.returned {
		t.Fatal("non-data evidence was conflated with eligibility or completion")
	}
	accept(7, temporalCancel)
	accept(7, temporalComplete)
	if life.Observe() || !evidence.done || !evidence.returned {
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
