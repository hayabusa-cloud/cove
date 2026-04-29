// ©Hayabusa Cloud Co., Ltd. 2026. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package cove_test

import (
	"testing"

	"code.hybscloud.com/cove"
)

func TestViewDuplicateExtractIdentity(t *testing.T) {
	v := cove.Observe(3, 7)
	if got := cove.Duplicate(v).Extract(); got != v {
		t.Fatalf("duplicate/extract identity: got %#v want %#v", got, v)
	}
}

func TestDuplicateViewAsk(t *testing.T) {
	v := cove.Observe(3, 7)
	dv := cove.Duplicate(v)
	if got := dv.Ask(); got != 3 {
		t.Fatalf("duplicate view ask: got %v want 3", got)
	}
}

func TestViewExtendExtractIdentity(t *testing.T) {
	v := cove.Observe(3, 7)
	got := cove.Extend(v, func(w cove.View[int, int]) int {
		return w.Extract()
	})
	if got != v {
		t.Fatalf("extend/extract identity: got %#v want %#v", got, v)
	}
}

func TestViewExtendAssociativity(t *testing.T) {
	v := cove.Observe(3, 7)
	f := func(w cove.View[int, int]) int {
		return w.Extract() + w.Ask()
	}
	g := func(w cove.View[int, int]) int {
		return w.Extract() * w.Ask()
	}

	left := cove.Extend(cove.Extend(v, f), g)
	right := cove.Extend(v, func(w cove.View[int, int]) int {
		return g(cove.Extend(w, f))
	})
	if left != right {
		t.Fatalf("extend associativity: got %#v want %#v", left, right)
	}
}

func TestViewDuplicateExtendCoherence(t *testing.T) {
	v := cove.Observe(3, 7)
	f := func(w cove.View[int, int]) int {
		return w.Extract() + w.Ask()
	}

	left := cove.Duplicate(cove.Extend(v, f))
	right := cove.Extend(v, func(w cove.View[int, int]) cove.View[int, int] {
		return cove.Extend(w, f)
	})
	if left != right {
		t.Fatalf("duplicate/extend coherence: got %#v want %#v", left, right)
	}
}

func TestCmdComposeUsesContextualExtension(t *testing.T) {
	v := cove.Observe(3, 7)
	f := cove.Cmd[int, int, int](func(w cove.View[int, int]) int {
		return w.Extract() + w.Ask()
	})
	g := cove.Cmd[int, int, int](func(w cove.View[int, int]) int {
		return w.Extract() * 2
	})

	if got, want := cove.Run(v, cove.Compose(g, f)), g(cove.Extend(v, f)); got != want {
		t.Fatalf("compose: got %v want %v", got, want)
	}
}

func TestCmdComposeAssociativity(t *testing.T) {
	v := cove.Observe(3, 7)
	f := cove.Cmd[int, int, int](func(w cove.View[int, int]) int {
		return w.Extract() + w.Ask()
	})
	g := cove.Cmd[int, int, int](func(w cove.View[int, int]) int {
		return w.Extract() * w.Ask()
	})
	h := cove.Cmd[int, int, int](func(w cove.View[int, int]) int {
		return w.Extract() - w.Ask()
	})

	left := cove.Run(v, cove.Compose(h, cove.Compose(g, f)))
	right := cove.Run(v, cove.Compose(cove.Compose(h, g), f))
	if left != right {
		t.Fatalf("compose associativity: got %v want %v", left, right)
	}
}

func TestCmdIdentityAndLift(t *testing.T) {
	v := cove.Observe(5, 4)
	id := cove.Cmd[int, int, int](cove.ExtractCmd[int, int])
	double := cove.LiftCmd[int, int, int](func(x int) int { return x * 2 })

	if got := cove.Run(v, id); got != 4 {
		t.Fatalf("identity command: got %v want 4", got)
	}
	if got := cove.Run(v, double); got != 8 {
		t.Fatalf("lifted command: got %v want 8", got)
	}
	if got := cove.Run(v, cove.Compose(double, id)); got != cove.Run(v, double) {
		t.Fatalf("right identity failed: got %v want %v", got, cove.Run(v, double))
	}
	if got := cove.Run(v, cove.Compose(id, double)); got != cove.Run(v, double) {
		t.Fatalf("left identity failed: got %v want %v", got, cove.Run(v, double))
	}
}

func TestMapReplaceWithContext(t *testing.T) {
	v := cove.Observe(1, 2)
	if got := cove.Map(v, func(x int) int { return x * 4 }); got != (cove.View[int, int]{Ctx: 1, Value: 8}) {
		t.Fatalf("map: got %#v", got)
	}
	if got := cove.MapContext(v, func(ctx int) string { return "ctx" }); got != (cove.View[string, int]{Ctx: "ctx", Value: 2}) {
		t.Fatalf("map context: got %#v", got)
	}
	if got := cove.Replace(v, 9); got != (cove.View[int, int]{Ctx: 1, Value: 9}) {
		t.Fatalf("replace: got %#v", got)
	}
	if got := cove.WithContext(v, 5); got != (cove.View[int, int]{Ctx: 5, Value: 2}) {
		t.Fatalf("with context: got %#v", got)
	}
}
