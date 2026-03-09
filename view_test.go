// ©Hayabusa Cloud Co., Ltd. 2026. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package cove_test

import (
	"code.hybscloud.com/cove"
	"testing"
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
