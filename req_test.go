// ©Hayabusa Cloud Co., Ltd. 2026. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package cove_test

import (
	"code.hybscloud.com/cove"
	"testing"
)

func TestNeedAllAnyNot(t *testing.T) {
	reqA := func(v int) bool { return v > 0 }
	reqB := func(v int) bool { return v%2 == 0 }
	reqBudget := func(v struct{ Budget int }) bool { return v.Budget > 0 }

	if !cove.Need(2, cove.All(reqA, reqB)) {
		t.Fatal("all should pass")
	}
	if cove.Need(1, cove.All(reqA, reqB)) {
		t.Fatal("all should fail")
	}
	if !cove.Need(1, cove.Any(reqA, reqB)) {
		t.Fatal("any should pass")
	}
	if !cove.Need(1, cove.Not(reqB)) {
		t.Fatal("not should pass")
	}
	if !cove.Need(10, nil) {
		t.Fatal("nil req should be true")
	}
	if !cove.Need(10, cove.True[int]()) {
		t.Fatal("true req should be true")
	}
	if cove.Need(10, cove.False[int]()) {
		t.Fatal("false req should be false")
	}
	if !cove.Need(struct{ Budget int }{Budget: 2}, cove.Pullback(reqA, func(v struct{ Budget int }) int { return v.Budget })) {
		t.Fatal("pullback should preserve requirement meaning")
	}
	if cove.Need(struct{ Budget int }{}, cove.Pullback(reqBudget, func(v struct{ Budget int }) struct{ Budget int } { return v })) {
		t.Fatal("pullback should fail when projected requirement fails")
	}
	if all := cove.All[int](nil, reqA, nil); cove.Need(1, all) != cove.Need(1, reqA) {
		t.Fatal("all should collapse nil requirements")
	}
	if any := cove.Any[int](nil, reqA, nil); !cove.Need(-1, any) {
		t.Fatal("any should treat nil requirements as satisfied terms")
	}
}

func TestAllEdgeCases(t *testing.T) {
	if got := cove.All[int](); !cove.Need(0, got) {
		t.Fatal("all zero should be true")
	}
	reqA := func(v int) bool { return v > 0 }
	if got := cove.All(reqA); !cove.Need(1, got) {
		t.Fatal("all single should pass")
	}
	if got := cove.All[int](nil, nil); !cove.Need(0, got) {
		t.Fatal("all nil-only should be true")
	}

	reqB := func(v int) bool { return v%2 == 0 }
	reqC := func(v int) bool { return v < 100 }
	three := cove.All(reqA, reqB, reqC)
	if !cove.Need(2, three) {
		t.Fatal("all three should pass for 2")
	}
	if cove.Need(-1, three) {
		t.Fatal("all three should fail for -1")
	}
	if cove.Need(3, three) {
		t.Fatal("all three should fail for 3 (not even)")
	}

	threeWithNils := cove.All[int](nil, reqA, nil, reqB, nil, reqC)
	if !cove.Need(2, threeWithNils) {
		t.Fatal("all three with nils should pass for 2")
	}
}

func TestAnyEdgeCases(t *testing.T) {
	if got := cove.Any[int](); cove.Need(0, got) {
		t.Fatal("any zero should be false")
	}
	reqA := func(v int) bool { return v > 0 }
	if got := cove.Any(reqA); !cove.Need(1, got) {
		t.Fatal("any single should pass")
	}
	if got := cove.Any[int](nil); !cove.Need(0, got) {
		t.Fatal("any single nil should be true")
	}
	if got := cove.Any[int](nil, nil); !cove.Need(0, got) {
		t.Fatal("any nil-only should be true")
	}

	reqB := func(v int) bool { return v%2 == 0 }
	reqC := func(v int) bool { return v == 7 }
	three := cove.Any(reqA, reqB, reqC)
	if !cove.Need(7, three) {
		t.Fatal("any three should pass for 7")
	}
	if !cove.Need(-2, three) {
		t.Fatal("any three should pass for -2 (even)")
	}
	if cove.Need(-3, three) {
		t.Fatal("any three should fail for -3")
	}

	threeWithNils := cove.Any[int](nil, reqA, nil, reqB, nil, reqC)
	if !cove.Need(-3, threeWithNils) {
		t.Fatal("any with nil terms should pass even when all non-nil terms fail")
	}
}

func TestPullbackNil(t *testing.T) {
	got := cove.Pullback[int, int](nil, func(v int) int { return v })
	if got != nil {
		t.Fatal("pullback nil should return nil")
	}
}

func TestNotNil(t *testing.T) {
	got := cove.Not[int](nil)
	if cove.Need(0, got) {
		t.Fatal("not nil should be false")
	}
}

func TestCheckRules(t *testing.T) {
	r1 := cove.Require("positive", func(v int) bool { return v > 0 })
	r2 := cove.Require("even", func(v int) bool { return v%2 == 0 })

	if report := cove.CheckRule(2, r1); !report.OK() || report.Checked != 1 {
		t.Fatalf("single rule success: %#v", report)
	}
	if report := cove.CheckRules(3, r1, r2); report.OK() || report.Failed != "even" || report.Checked != 2 {
		t.Fatalf("rule failure: %#v", report)
	}
	type runtime struct{ Budget int }
	pulled := cove.PullbackRule(r1, func(rt runtime) int { return rt.Budget })
	if report := cove.CheckRule(runtime{Budget: 4}, pulled); !report.OK() {
		t.Fatalf("pulled rule should pass: %#v", report)
	}
}

func TestRuleReqAndMatch(t *testing.T) {
	r := cove.Require("positive", func(v int) bool { return v > 0 })
	req := r.Req()
	if !cove.Need(1, req) {
		t.Fatal("rule.Req should return the underlying requirement")
	}
	if !r.Match(1) {
		t.Fatal("rule.Match should pass for 1")
	}
	if r.Match(-1) {
		t.Fatal("rule.Match should fail for -1")
	}
}

func TestCheckRulesAllPass(t *testing.T) {
	r1 := cove.Require("positive", func(v int) bool { return v > 0 })
	r2 := cove.Require("even", func(v int) bool { return v%2 == 0 })
	report := cove.CheckRules(2, r1, r2)
	if !report.OK() || report.Checked != 2 {
		t.Fatalf("all rules should pass: %#v", report)
	}
}

func TestCheckRulesEmpty(t *testing.T) {
	report := cove.CheckRules[int](0)
	if !report.OK() || report.Checked != 0 {
		t.Fatalf("empty rules should pass: %#v", report)
	}
}
