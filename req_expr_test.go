// ©Hayabusa Cloud Co., Ltd. 2026. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package cove_test

import (
	"code.hybscloud.com/cove"
	"testing"
)

func TestExprReqBasics(t *testing.T) {
	if !cove.NeedExpr(0, cove.ExprTrue[int]()) {
		t.Fatal("ExprTrue should be true")
	}
	if cove.NeedExpr(0, cove.ExprFalse[int]()) {
		t.Fatal("ExprFalse should be false")
	}
	atom := cove.ExprAtom(func(v int) bool { return v > 0 })
	if !cove.NeedExpr(1, atom) {
		t.Fatal("ExprAtom should pass for 1")
	}
	if cove.NeedExpr(-1, atom) {
		t.Fatal("ExprAtom should fail for -1")
	}
}

func TestExprReqZeroValue(t *testing.T) {
	var zero cove.ReqExpr[int]
	if !cove.NeedExpr(0, zero) {
		t.Fatal("zero ReqExpr should be true")
	}
}

func TestExprNot(t *testing.T) {
	atom := cove.ExprAtom(func(v int) bool { return v > 0 })
	neg := cove.ExprNot(atom)
	if cove.NeedExpr(1, neg) {
		t.Fatal("ExprNot should negate passing")
	}
	if !cove.NeedExpr(-1, neg) {
		t.Fatal("ExprNot should negate failing")
	}
}

func TestExprAll(t *testing.T) {
	if !cove.NeedExpr(0, cove.ExprAll[int]()) {
		t.Fatal("ExprAll() should be true")
	}
	atom := cove.ExprAtom(func(v int) bool { return v > 0 })
	if !cove.NeedExpr(1, cove.ExprAll(atom)) {
		t.Fatal("ExprAll single should pass")
	}
	atomB := cove.ExprAtom(func(v int) bool { return v%2 == 0 })
	all2 := cove.ExprAll(atom, atomB)
	if !cove.NeedExpr(2, all2) {
		t.Fatal("ExprAll(>0, even) should pass for 2")
	}
	if cove.NeedExpr(1, all2) {
		t.Fatal("ExprAll(>0, even) should fail for 1")
	}
	atomC := cove.ExprAtom(func(v int) bool { return v < 100 })
	all3 := cove.ExprAll(atom, atomB, atomC)
	if !cove.NeedExpr(2, all3) {
		t.Fatal("ExprAll 3 should pass for 2")
	}
	if cove.NeedExpr(-1, all3) {
		t.Fatal("ExprAll 3 should fail for -1")
	}
}

func TestExprAny(t *testing.T) {
	if cove.NeedExpr(0, cove.ExprAny[int]()) {
		t.Fatal("ExprAny() should be false")
	}
	atom := cove.ExprAtom(func(v int) bool { return v > 0 })
	if !cove.NeedExpr(1, cove.ExprAny(atom)) {
		t.Fatal("ExprAny single should pass")
	}
	atomB := cove.ExprAtom(func(v int) bool { return v%2 == 0 })
	any2 := cove.ExprAny(atom, atomB)
	if !cove.NeedExpr(1, any2) {
		t.Fatal("ExprAny(>0, even) should pass for 1")
	}
	if cove.NeedExpr(-3, any2) {
		t.Fatal("ExprAny(>0, even) should fail for -3")
	}
	atomC := cove.ExprAtom(func(v int) bool { return v == 7 })
	any3 := cove.ExprAny(atom, atomB, atomC)
	if !cove.NeedExpr(7, any3) {
		t.Fatal("ExprAny 3 should pass for 7")
	}
	if cove.NeedExpr(-3, any3) {
		t.Fatal("ExprAny 3 should fail for -3")
	}
}

func TestExprPullback(t *testing.T) {
	type runtime struct{ Budget int }
	atom := cove.ExprAtom(func(v int) bool { return v > 0 })
	pulled := cove.ExprPullback(atom, func(rt runtime) int { return rt.Budget })
	if !cove.NeedExpr(runtime{Budget: 2}, pulled) {
		t.Fatal("ExprPullback should pass")
	}
	if cove.NeedExpr(runtime{Budget: 0}, pulled) {
		t.Fatal("ExprPullback should fail")
	}
	pulledTrue := cove.ExprPullback(cove.ExprTrue[int](), func(rt runtime) int { return rt.Budget })
	if !cove.NeedExpr(runtime{}, pulledTrue) {
		t.Fatal("ExprPullback true shortcut")
	}
	pulledFalse := cove.ExprPullback(cove.ExprFalse[int](), func(rt runtime) int { return rt.Budget })
	if cove.NeedExpr(runtime{}, pulledFalse) {
		t.Fatal("ExprPullback false shortcut")
	}
}

func TestExprPullbackComposition(t *testing.T) {
	type inner struct{ Value int }
	type outer struct{ Inner inner }
	atom := cove.ExprAtom(func(v int) bool { return v > 0 })
	step1 := cove.ExprPullback(atom, func(i inner) int { return i.Value })
	step2 := cove.ExprPullback(step1, func(o outer) inner { return o.Inner })
	if !cove.NeedExpr(outer{Inner: inner{Value: 5}}, step2) {
		t.Fatal("double pullback should pass for positive inner")
	}
	if cove.NeedExpr(outer{Inner: inner{Value: -1}}, step2) {
		t.Fatal("double pullback should fail for negative inner")
	}
}

func TestExprPullbackComposite(t *testing.T) {
	type runtime struct{ Budget int }
	expr := cove.ExprAll(
		cove.ExprAny(
			cove.ExprAtom(func(v int) bool { return v > 0 }),
			cove.ExprAtom(func(v int) bool { return v == -4 }),
		),
		cove.ExprNot(cove.ExprAtom(func(v int) bool { return v == 7 })),
	)
	pulled := cove.ExprPullback(expr, func(rt runtime) int { return rt.Budget })
	if !cove.NeedExpr(runtime{Budget: 2}, pulled) {
		t.Fatal("composite pullback should pass for positive budget")
	}
	if !cove.NeedExpr(runtime{Budget: -4}, pulled) {
		t.Fatal("composite pullback should pass for alternate branch")
	}
	if cove.NeedExpr(runtime{Budget: 7}, pulled) {
		t.Fatal("composite pullback should fail when negated branch matches")
	}
	if cove.NeedExpr(runtime{Budget: 0}, pulled) {
		t.Fatal("composite pullback should fail when all branches fail")
	}
}

func TestRuleExprBasics(t *testing.T) {
	atom := cove.ExprAtom(func(v int) bool { return v > 0 })
	rule := cove.RequireExpr("positive", atom)
	if rule.Name != "positive" {
		t.Fatalf("unexpected name: %s", rule.Name)
	}
	if !cove.NeedExpr(1, rule.Req()) {
		t.Fatal("Req() should return a passing requirement for 1")
	}
	if !rule.Match(1) {
		t.Fatal("Match should pass for 1")
	}
	if rule.Match(-1) {
		t.Fatal("Match should fail for -1")
	}
}

func TestCheckRuleExpr(t *testing.T) {
	atom := cove.ExprAtom(func(v int) bool { return v > 0 })
	rule := cove.RequireExpr("positive", atom)
	if report := cove.CheckRuleExpr(2, rule); !report.OK() || report.Checked != 1 {
		t.Fatalf("single rule success: %#v", report)
	}
	if report := cove.CheckRuleExpr(-1, rule); report.OK() || report.Failed != "positive" || report.Checked != 1 {
		t.Fatalf("single rule failure: %#v", report)
	}
}

func TestCheckRulesExpr(t *testing.T) {
	r1 := cove.RequireExpr("positive", cove.ExprAtom(func(v int) bool { return v > 0 }))
	r2 := cove.RequireExpr("even", cove.ExprAtom(func(v int) bool { return v%2 == 0 }))
	if report := cove.CheckRulesExpr(2, r1, r2); !report.OK() || report.Checked != 2 {
		t.Fatalf("all pass: %#v", report)
	}
	if report := cove.CheckRulesExpr(3, r1, r2); report.OK() || report.Failed != "even" || report.Checked != 2 {
		t.Fatalf("second fails: %#v", report)
	}
	if report := cove.CheckRulesExpr(-1, r1, r2); report.OK() || report.Failed != "positive" || report.Checked != 1 {
		t.Fatalf("first fails: %#v", report)
	}
	if report := cove.CheckRulesExpr[int](0); !report.OK() || report.Checked != 0 {
		t.Fatalf("empty pass: %#v", report)
	}
}

func TestPullbackRuleExpr(t *testing.T) {
	type runtime struct{ Budget int }
	atom := cove.ExprAtom(func(v int) bool { return v > 0 })
	rule := cove.RequireExpr("positive", atom)
	pulled := cove.PullbackRuleExpr(rule, func(rt runtime) int { return rt.Budget })
	if pulled.Name != "positive" {
		t.Fatalf("name should be preserved: %s", pulled.Name)
	}
	if report := cove.CheckRuleExpr(runtime{Budget: 2}, pulled); !report.OK() {
		t.Fatalf("pulled rule should pass: %#v", report)
	}
	if report := cove.CheckRuleExpr(runtime{Budget: 0}, pulled); report.OK() {
		t.Fatalf("pulled rule should fail: %#v", report)
	}
}
