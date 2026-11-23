// ©Hayabusa Cloud Co., Ltd. 2026. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package cove_test

import (
	"testing"

	"code.hybscloud.com/cove"
	"code.hybscloud.com/kont"
)

func TestSteppingBridge(t *testing.T) {
	cont := kont.Bind(kont.Perform(ping{}), func(v int) kont.Eff[int] {
		return kont.Pure(v + 1)
	})

	if _, susp := cove.Step(cont); susp == nil {
		t.Fatal("expected cont-world suspension")
	} else {
		susp.Discard()
	}

	expr := cove.Reify(cont)
	reflectedCopy := cove.Reflect(expr)
	_, step := cove.StepExprWith("ctx", expr)
	if step.Suspension == nil {
		t.Fatal("expected expr-world suspension")
	}
	if step.Ask() != "ctx" {
		t.Fatalf("unexpected step ctx: %#v", step)
	}
	if _, ok := step.Op().(ping); !ok {
		t.Fatalf("unexpected step op type: %#v", step.Op())
	}
	expectPanic(t, func() { cove.Step(reflectedCopy) })
	guarded := cove.Guard(func(v int) bool { return v > 0 }, step.Extract())
	if !guarded.Check(1) {
		t.Fatal("expected guarded suspension success")
	}
	checked, ok := cove.CheckSuspension(1, step.Extract(), func(v int) bool { return v > 0 })
	if !ok || checked.Op() == nil {
		t.Fatalf("unexpected checked suspension: %#v %v", checked, ok)
	}
	result, next := checked.Resume(41)
	if next.Suspension != nil {
		t.Fatalf("expected final completion, got next=%#v", next)
	}
	if result != 42 {
		t.Fatalf("unexpected result: %d", result)
	}

	fresh := cove.Reify(cont)
	_, susp := cove.Step(cove.Reflect(fresh))
	if susp == nil {
		t.Fatal("expected reflected cont-world suspension")
	}
	if _, ok := susp.Op().(ping); !ok {
		t.Fatalf("unexpected reflected op type: %#v", susp.Op())
	}
	expectPanic(t, func() { cove.StepExpr(fresh) })
	if result, next := susp.Resume(9); result != 10 || next != nil {
		t.Fatalf("reflected run returned %d, %v", result, next)
	}
}

func TestReifyCompleted(t *testing.T) {
	calls := 0
	m := func(k func(int) kont.Resumed) kont.Resumed {
		calls++
		return k(42)
	}
	expr := cove.Reify(m)
	if calls != 1 {
		t.Fatalf("Reify ran the computation %d times, want 1", calls)
	}
	for range 2 {
		if result, next := cove.StepExpr(expr); result != 42 || next != nil || calls != 1 {
			t.Fatalf("StepExpr returned %d, %v after %d evaluations", result, next, calls)
		}
	}
}

func TestReflectReq(t *testing.T) {
	atom := cove.ExprAtom(func(v int) bool { return v > 0 })
	all := cove.ExprAll(atom, cove.ExprAtom(func(v int) bool { return v%2 == 0 }))
	reflected := cove.ReflectReq(all)
	if !cove.Need(2, reflected) {
		t.Fatal("ReflectReq should pass for 2")
	}
	if cove.Need(1, reflected) {
		t.Fatal("ReflectReq should fail for 1")
	}
}

func TestReflectReqLaw(t *testing.T) {
	expr := cove.ExprAll(
		cove.ExprAtom(func(v int) bool { return v > 0 }),
		cove.ExprNot(cove.ExprAtom(func(v int) bool { return v > 100 })),
	)
	reflected := cove.ReflectReq(expr)
	for _, v := range []int{-1, 0, 1, 50, 100, 101} {
		got := cove.Need(v, reflected)
		want := cove.NeedExpr(v, expr)
		if got != want {
			t.Fatalf("ReflectReq law violation at %d: Need=%v NeedExpr=%v", v, got, want)
		}
	}
}

func TestReifyReq(t *testing.T) {
	req := cove.Req[int](func(v int) bool { return v > 0 && v%2 == 0 })
	reified := cove.ReifyReq(req)
	if !cove.NeedExpr(2, reified) {
		t.Fatal("ReifyReq should pass for 2")
	}
	if cove.NeedExpr(1, reified) {
		t.Fatal("ReifyReq should fail for 1")
	}
}

func TestReifyReqLaw(t *testing.T) {
	req := cove.Req[int](func(v int) bool { return v > 0 && v < 100 })
	reified := cove.ReifyReq(req)
	for _, v := range []int{-1, 0, 1, 50, 100, 101} {
		got := cove.NeedExpr(v, reified)
		want := cove.Need(v, req)
		if got != want {
			t.Fatalf("ReifyReq law violation at %d: NeedExpr=%v Need=%v", v, got, want)
		}
	}
}
