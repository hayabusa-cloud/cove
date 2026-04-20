// ©Hayabusa Cloud Co., Ltd. 2026. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package cove_test

import (
	"testing"

	"code.hybscloud.com/cove"
)

func TestCheckedExprIntoView(t *testing.T) {
	atom := cove.ExprAtom(func(v int) bool { return v > 0 })
	checked := cove.GuardExpr(atom, "ok")
	view, ok := checked.IntoView(2)
	if !ok {
		t.Fatal("expected checked expr view")
	}
	if view.Ask() != 2 || view.Extract() != "ok" {
		t.Fatalf("unexpected view: %#v", view)
	}
	if _, ok := checked.IntoView(0); ok {
		t.Fatal("expected checked expr failure")
	}
}

func TestCheckedExprCheck(t *testing.T) {
	atom := cove.ExprAtom(func(v int) bool { return v > 0 })
	checked := cove.GuardExpr(atom, "ok")
	if !checked.Check(1) {
		t.Fatal("Check should pass")
	}
	if checked.Check(-1) {
		t.Fatal("Check should fail")
	}
}

func TestCheckedExprMustViewSuccess(t *testing.T) {
	atom := cove.ExprAtom(func(v int) bool { return v > 0 })
	checked := cove.GuardExpr(atom, "ok")
	v := checked.MustView(1)
	if v.Extract() != "ok" || v.Ask() != 1 {
		t.Fatalf("unexpected must view: %#v", v)
	}
}

func TestCheckedExprMustViewPanics(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic from MustView")
		}
		if s, ok := r.(string); !ok || s != "cove: requirement does not hold" {
			t.Fatalf("unexpected panic value: %v", r)
		}
	}()
	atom := cove.ExprAtom(func(v int) bool { return v > 0 })
	checked := cove.GuardExpr(atom, "ok")
	checked.MustView(0)
}

func TestMapCheckedExpr(t *testing.T) {
	atom := cove.ExprAtom(func(v int) bool { return v > 0 })
	checked := cove.GuardExpr(atom, "ok")
	mapped := cove.MapCheckedExpr(checked, func(s string) int { return len(s) })
	if mapped.Value != 2 || !mapped.Check(3) {
		t.Fatalf("unexpected mapped: %#v", mapped)
	}
}

func TestPullbackCheckedExpr(t *testing.T) {
	type runtime struct{ Budget int }
	atom := cove.ExprAtom(func(v int) bool { return v > 0 })
	checked := cove.GuardExpr(atom, "ok")
	pulled := cove.PullbackCheckedExpr(checked, func(rt runtime) int { return rt.Budget })
	if _, ok := pulled.IntoView(runtime{Budget: 2}); !ok {
		t.Fatal("expected pulled checked expr view")
	}
	if _, ok := pulled.IntoView(runtime{Budget: 0}); ok {
		t.Fatal("expected pulled checked expr failure")
	}
}

func TestGuardedExprIntoView(t *testing.T) {
	rule := cove.RequireExpr("positive", cove.ExprAtom(func(v int) bool { return v > 0 }))
	guarded := cove.GuardRuleExpr(rule, 7)
	view, report := guarded.IntoView(1)
	if !report.OK() || view.Extract() != 7 || view.Ask() != 1 {
		t.Fatalf("unexpected guarded success: view=%#v report=%#v", view, report)
	}
	_, report = guarded.IntoView(0)
	if report.OK() || report.Failed != "positive" {
		t.Fatalf("unexpected guarded failure: %#v", report)
	}
}

func TestGuardedExprCheck(t *testing.T) {
	rule := cove.RequireExpr("positive", cove.ExprAtom(func(v int) bool { return v > 0 }))
	guarded := cove.GuardRuleExpr(rule, 7)
	report := guarded.Check(1)
	if !report.OK() {
		t.Fatalf("guarded check should pass: %#v", report)
	}
	report = guarded.Check(0)
	if report.OK() || report.Failed != "positive" {
		t.Fatalf("guarded check should fail: %#v", report)
	}
}

func TestMapGuardedExpr(t *testing.T) {
	rule := cove.RequireExpr("positive", cove.ExprAtom(func(v int) bool { return v > 0 }))
	guarded := cove.GuardRuleExpr(rule, 7)
	mapped := cove.MapGuardedExpr(guarded, func(v int) string { return "x" })
	if mapped.Value != "x" || mapped.Rule.Name != "positive" {
		t.Fatalf("unexpected mapped guarded: %#v", mapped)
	}
}

func TestPullbackGuardedExpr(t *testing.T) {
	type runtime struct{ Budget int }
	rule := cove.RequireExpr("positive", cove.ExprAtom(func(v int) bool { return v > 0 }))
	guarded := cove.GuardRuleExpr(rule, 7)
	pulled := cove.PullbackGuardedExpr(guarded, func(rt runtime) int { return rt.Budget })
	if pulled.Rule.Name != "positive" {
		t.Fatalf("name should be preserved: %s", pulled.Rule.Name)
	}
	if _, report := pulled.IntoView(runtime{Budget: 1}); !report.OK() {
		t.Fatalf("unexpected pulled guarded failure: %#v", report)
	}
	if _, report := pulled.IntoView(runtime{Budget: 0}); report.OK() {
		t.Fatalf("expected pulled guarded failure")
	}
}

func TestGuardedExprIntoViewUnnamedFailureDoesNotExposeValue(t *testing.T) {
	guarded := cove.GuardRuleExpr(cove.RuleExpr[int]{Check: cove.ExprAtom(func(v int) bool { return v > 0 })}, "secret")
	view, report := guarded.IntoView(0)
	if report.OK() {
		t.Fatalf("unnamed failing guard must not pass: %#v", report)
	}
	if report.Failed == "" {
		t.Fatalf("unnamed failing guard must report a failure label: %#v", report)
	}
	if got := view.Extract(); got != "" {
		t.Fatalf("guarded failure exposed value %q", got)
	}
}
