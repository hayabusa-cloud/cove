// ©Hayabusa Cloud Co., Ltd. 2026. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package cove_test

import (
	"code.hybscloud.com/cove"
	"code.hybscloud.com/kont"
	"testing"
)

func TestCheckSuspensionFailure(t *testing.T) {
	cont := kont.Perform(ping{})
	expr := cove.Reify(cont)
	_, susp := cove.StepExpr(expr)
	if susp == nil {
		t.Fatal("expected suspension")
	}
	_, ok := cove.CheckSuspension(0, susp, func(v int) bool { return v > 0 })
	if ok {
		t.Fatal("expected check suspension failure")
	}
	susp.Discard()
}

func TestSuspensionViewDiscard(t *testing.T) {
	cont := kont.Perform(ping{})
	expr := cove.Reify(cont)
	_, step := cove.StepExprWith("ctx", expr)
	if step.Suspension == nil {
		t.Fatal("expected suspension")
	}
	step.Discard()
}

func TestStepWithContCompletion(t *testing.T) {
	cont := kont.Pure[kont.Resumed](42)
	result, sv := cove.StepWith("ctx", cont)
	if sv.Suspension != nil {
		t.Fatal("expected completion")
	}
	if result != 42 {
		t.Fatalf("unexpected result: %d", result)
	}
}

func TestStepWithContSuspension(t *testing.T) {
	cont := kont.Bind(kont.Perform(ping{}), func(v int) kont.Eff[int] {
		return kont.Pure(v + 1)
	})
	_, step := cove.StepWith("ctx", cont)
	if step.Suspension == nil {
		t.Fatal("expected suspension")
	}
	if step.Ask() != "ctx" {
		t.Fatalf("unexpected ctx: %v", step.Ask())
	}
	result, sv := step.Resume(99)
	if sv.Suspension != nil {
		t.Fatal("expected completion after resume")
	}
	if result != 100 {
		t.Fatalf("unexpected result: %d", result)
	}
}

func TestSuspensionViewResumeChain(t *testing.T) {
	cont := kont.Bind(kont.Perform(ping{}), func(v int) kont.Eff[int] {
		return kont.Bind(kont.Perform(ping{}), func(w int) kont.Eff[int] {
			return kont.Pure(v + w)
		})
	})
	expr := cove.Reify(cont)
	_, step := cove.StepExprWith("ctx", expr)
	if step.Suspension == nil {
		t.Fatal("expected first suspension")
	}
	_, step2 := step.Resume(10)
	if step2.Suspension == nil {
		t.Fatal("expected second suspension after resume")
	}
	if step2.Ask() != "ctx" {
		t.Fatalf("unexpected ctx after chained resume: %v", step2.Ask())
	}
	result, sv := step2.Resume(20)
	if sv.Suspension != nil {
		t.Fatal("expected completion")
	}
	if result != 30 {
		t.Fatalf("unexpected result: %d", result)
	}
}

func TestStepExprWithCompletion(t *testing.T) {
	expr := kont.ExprReturn[int](42)
	result, sv := cove.StepExprWith("ctx", expr)
	if sv.Suspension != nil {
		t.Fatal("expected completion")
	}
	if result != 42 {
		t.Fatalf("unexpected result: %d", result)
	}
}

func TestResumeWith(t *testing.T) {
	type budget struct{ N int }
	cont := kont.Bind(kont.Perform(ping{}), func(v int) kont.Eff[int] {
		return kont.Bind(kont.Perform(ping{}), func(w int) kont.Eff[int] {
			return kont.Pure(v + w)
		})
	})
	expr := cove.Reify(cont)
	_, step := cove.StepExprWith(budget{N: 10}, expr)
	if step.Suspension == nil {
		t.Fatal("expected first suspension")
	}
	if step.Ask().N != 10 {
		t.Fatalf("unexpected initial budget: %d", step.Ask().N)
	}
	decrement := func(b budget) budget { return budget{N: b.N - 1} }
	_, step2 := step.ResumeWith(10, decrement)
	if step2.Suspension == nil {
		t.Fatal("expected second suspension")
	}
	if step2.Ask().N != 9 {
		t.Fatalf("budget should have decremented: got %d want 9", step2.Ask().N)
	}
	result, sv := step2.ResumeWith(20, decrement)
	if sv.Suspension != nil {
		t.Fatal("expected completion")
	}
	if result != 30 {
		t.Fatalf("unexpected result: %d", result)
	}
}

func TestResumeWithCompletion(t *testing.T) {
	cont := kont.Perform(ping{})
	expr := cove.Reify(cont)
	_, step := cove.StepExprWith(42, expr)
	if step.Suspension == nil {
		t.Fatal("expected suspension")
	}
	result, sv := step.ResumeWith(7, func(v int) int { return v + 1 })
	if sv.Suspension != nil {
		t.Fatal("expected completion")
	}
	if result != 7 {
		t.Fatalf("unexpected result: %d", result)
	}
}

func TestCompletedSuspensionViewOpPanics(t *testing.T) {
	_, step := cove.StepWith("ctx", kont.Pure[kont.Resumed](42))
	expectPanicMessage(t, "cove: suspension view completed", func() {
		_ = step.Op()
	})
}

func TestCompletedSuspensionViewResumePanics(t *testing.T) {
	_, step := cove.StepWith("ctx", kont.Pure[kont.Resumed](42))
	expectPanicMessage(t, "cove: suspension view completed", func() {
		step.Resume(7)
	})
}

func TestCompletedSuspensionViewResumeWithPanics(t *testing.T) {
	_, step := cove.StepWith(42, kont.Pure[kont.Resumed](7))
	expectPanicMessage(t, "cove: suspension view completed", func() {
		step.ResumeWith(9, func(v int) int { return v + 1 })
	})
}

func TestCompletedSuspensionViewDiscardPanics(t *testing.T) {
	_, step := cove.StepWith("ctx", kont.Pure[kont.Resumed](42))
	expectPanicMessage(t, "cove: suspension view completed", func() {
		step.Discard()
	})
}

func TestCheckSuspensionExpr(t *testing.T) {
	cont := kont.Perform(ping{})
	expr := cove.Reify(cont)
	_, susp := cove.StepExpr(expr)
	if susp == nil {
		t.Fatal("expected suspension")
	}
	atom := cove.ExprAtom(func(v int) bool { return v > 0 })
	sv, ok := cove.CheckSuspensionExpr(1, susp, atom)
	if !ok || sv.Suspension == nil {
		t.Fatalf("expected check suspension success: ok=%v", ok)
	}
	sv.Discard()
}

func TestCheckSuspensionExprFailure(t *testing.T) {
	cont := kont.Perform(ping{})
	expr := cove.Reify(cont)
	_, susp := cove.StepExpr(expr)
	if susp == nil {
		t.Fatal("expected suspension")
	}
	atom := cove.ExprAtom(func(v int) bool { return v > 0 })
	_, ok := cove.CheckSuspensionExpr(0, susp, atom)
	if ok {
		t.Fatal("expected check suspension failure")
	}
	susp.Discard()
}
