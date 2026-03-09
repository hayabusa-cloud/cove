// ©Hayabusa Cloud Co., Ltd. 2026. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package cove_test

import (
	"code.hybscloud.com/cove"
	"testing"
)

func TestCheckedIntoView(t *testing.T) {
	checked := cove.Guard(func(v int) bool { return v > 0 }, "ok")
	view, ok := checked.IntoView(2)
	if !ok {
		t.Fatal("expected checked view")
	}
	if view.Ask() != 2 || view.Extract() != "ok" {
		t.Fatalf("unexpected view: %#v", view)
	}
	if _, ok := checked.IntoView(0); ok {
		t.Fatal("expected checked view failure")
	}
	if mapped := cove.MapChecked(checked, func(s string) int { return len(s) }); mapped.Value != 2 || !mapped.Check(3) {
		t.Fatalf("unexpected mapped checked: %#v", mapped)
	}
	type runtime struct{ Budget int }
	pulled := cove.PullbackChecked(checked, func(rt runtime) int { return rt.Budget })
	if _, ok := pulled.IntoView(runtime{Budget: 2}); !ok {
		t.Fatal("expected pulled checked view")
	}
}

func TestCheckedMustViewSuccess(t *testing.T) {
	checked := cove.Guard(func(v int) bool { return v > 0 }, "ok")
	v := checked.MustView(1)
	if v.Extract() != "ok" || v.Ask() != 1 {
		t.Fatalf("unexpected must view: %#v", v)
	}
}

func TestCheckedMustViewPanics(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic from MustView")
		}
		if s, ok := r.(string); !ok || s != "cove: requirement does not hold" {
			t.Fatalf("unexpected panic value: %v", r)
		}
	}()
	checked := cove.Guard(func(v int) bool { return v > 0 }, "ok")
	checked.MustView(0)
}

func TestGuardedIntoView(t *testing.T) {
	guarded := cove.GuardRule(cove.Require("positive", func(v int) bool { return v > 0 }), 7)
	view, report := guarded.IntoView(1)
	if !report.OK() || view.Extract() != 7 || view.Ask() != 1 {
		t.Fatalf("unexpected guarded success: view=%#v report=%#v", view, report)
	}
	_, report = guarded.IntoView(0)
	if report.OK() || report.Failed != "positive" {
		t.Fatalf("unexpected guarded failure: %#v", report)
	}
	mapped := cove.MapGuarded(guarded, func(v int) string { return "x" })
	if mapped.Value != "x" || mapped.Rule.Name != "positive" {
		t.Fatalf("unexpected mapped guarded: %#v", mapped)
	}
	type runtime struct{ Budget int }
	pulled := cove.PullbackGuarded(guarded, func(rt runtime) int { return rt.Budget })
	if _, report := pulled.IntoView(runtime{Budget: 1}); !report.OK() {
		t.Fatalf("unexpected pulled guarded failure: %#v", report)
	}
}

func TestGuardedCheck(t *testing.T) {
	guarded := cove.GuardRule(cove.Require("positive", func(v int) bool { return v > 0 }), 7)
	report := guarded.Check(1)
	if !report.OK() {
		t.Fatalf("guarded check should pass: %#v", report)
	}
	report = guarded.Check(0)
	if report.OK() || report.Failed != "positive" {
		t.Fatalf("guarded check should fail: %#v", report)
	}
}
