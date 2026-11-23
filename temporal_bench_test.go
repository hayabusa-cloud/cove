// ©Hayabusa Cloud Co., Ltd. 2026. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package cove_test

import (
	"testing"

	"code.hybscloud.com/cove"
	"code.hybscloud.com/kont"
)

var temporalBoolSink bool
var temporalIntSink int
var temporalReqSink cove.Req[temporalContext]
var temporalExprSink cove.ReqExpr[temporalContext]

func temporalDirect(c temporalContext) bool { return c.eligible && c.generation == 7 }

func BenchmarkTemporalObservation(b *testing.B) {
	b.Run("Direct", func(b *testing.B) {
		b.ReportAllocs()
		for i := range b.N {
			c := temporalContext{i&1 != 0, uint64(i & 7)}
			temporalBoolSink = temporalDirect(c)
		}
	})
	b.Run("Cove", func(b *testing.B) {
		b.ReportAllocs()
		for i := range b.N {
			c := temporalContext{i&1 != 0, uint64(i & 7)}
			temporalBoolSink = cove.Need(cove.Observe(c, 0).Ask(), temporalDirect)
		}
	})
	generation := cove.ExprAtom(func(g uint64) bool { return g == 7 })
	for _, tc := range []struct {
		name  string
		guard cove.ReqExpr[temporalContext]
	}{
		{"ReqExpr", cove.ExprAtom(func(c temporalContext) bool { return c.generation == 7 })},
		{"ExprPullback", cove.ExprPullback(generation, func(c temporalContext) uint64 { return c.generation })},
	} {
		b.Run(tc.name, func(b *testing.B) {
			req := cove.ExprAll(cove.ExprAtom(temporalAllowed), tc.guard)
			b.ReportAllocs()
			b.ResetTimer()
			for i := range b.N {
				temporalBoolSink = cove.NeedExpr(temporalContext{i&1 != 0, uint64(i & 7)}, req)
			}
		})
	}
}

func BenchmarkTemporalPullbackSetup(b *testing.B) {
	project := func(c temporalContext) bool { return c.eligible }
	predicate := func(v bool) bool { return v }
	b.Run("Req", func(b *testing.B) {
		b.ReportAllocs()
		for range b.N {
			temporalReqSink = cove.Pullback(predicate, project)
		}
	})
	b.Run("ReqExpr", func(b *testing.B) {
		req := cove.ExprAll(cove.ExprAtom(predicate), cove.ExprNot(cove.ExprFalse[bool]()))
		b.ReportAllocs()
		b.ResetTimer()
		for range b.N {
			temporalExprSink = cove.ExprPullback(req, project)
		}
	})
}

// These measure complete activations, including fresh kont authority. They do
// not attribute continuation provisioning to the context wrapper alone.
func BenchmarkTemporalResume(b *testing.B) {
	b.Run("Direct", func(b *testing.B) {
		b.ReportAllocs()
		for range b.N {
			_, owner := kont.Step(kont.Perform(temporalValue{input: 1}))
			if temporalDirect(temporalContext{true, 7}) {
				result, _ := owner.Resume(2)
				temporalIntSink = result
			} else {
				owner.Discard()
			}
		}
	})
	b.Run("Cove", func(b *testing.B) {
		b.ReportAllocs()
		for range b.N {
			_, owner := cove.StepWith(temporalContext{true, 7}, kont.Perform(temporalValue{input: 1}))
			checked, ok := cove.CheckSuspension(owner.Ctx, owner.Suspension, temporalDirect)
			if ok {
				result, _ := checked.Resume(2)
				temporalIntSink = result
			} else {
				owner.Discard()
			}
		}
	})
}

func BenchmarkTemporalSuccessor(b *testing.B) {
	program := kont.Bind(kont.Perform(temporalValue{input: 1}), func(v int) kont.Eff[int] {
		return kont.Perform(temporalValue{input: v})
	})
	b.Run("Direct", func(b *testing.B) {
		b.ReportAllocs()
		for range b.N {
			_, owner := kont.Step(program)
			_, next := owner.Resume(2)
			result, _ := next.Resume(3)
			temporalIntSink = result
		}
	})
	b.Run("Cove", func(b *testing.B) {
		b.ReportAllocs()
		for range b.N {
			_, owner := cove.StepWith(7, program)
			_, next := owner.Resume(2)
			result, _ := next.Resume(3)
			temporalIntSink = result
		}
	})
}
