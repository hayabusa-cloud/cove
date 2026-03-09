// ©Hayabusa Cloud Co., Ltd. 2026. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package cove_test

import (
	"code.hybscloud.com/cove"
	"code.hybscloud.com/kont"
	"testing"
)

func TestSteppingBridge(t *testing.T) {
	cont := kont.Bind(kont.Perform(ping{}), func(v int) kont.Eff[int] {
		return kont.Pure(v + 1)
	})

	if _, susp := cove.Step(cont); susp == nil {
		t.Fatal("expected cont-world suspension")
	}

	expr := cove.Reify(cont)
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
	result, next := step.Resume(41)
	if next.Suspension != nil {
		t.Fatalf("expected final completion, got next=%#v", next)
	}
	if result != 42 {
		t.Fatalf("unexpected result: %d", result)
	}

	reflected := cove.Reflect(expr)
	if _, susp := cove.Step(reflected); susp == nil {
		t.Fatal("expected reflected cont-world suspension")
	} else if _, ok := susp.Op().(ping); !ok {
		t.Fatalf("unexpected reflected op type: %#v", susp.Op())
	}

	guarded := cove.Guard(func(v int) bool { return v > 0 }, step.Extract())
	if !guarded.Check(1) {
		t.Fatal("expected guarded suspension success")
	}
	checked, ok := cove.CheckSuspension(1, step.Extract(), func(v int) bool { return v > 0 })
	if !ok || checked.Op() == nil {
		t.Fatalf("unexpected checked suspension: %#v %v", checked, ok)
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
