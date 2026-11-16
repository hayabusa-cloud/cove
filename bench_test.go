// ©Hayabusa Cloud Co., Ltd. 2026. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package cove_test

import (
	"testing"

	"code.hybscloud.com/cove"
	"code.hybscloud.com/kont"
)

var boolSink bool
var intSink int
var viewSink cove.View[int, int]

var requirementInputs = [...]int{-1, 0, 1, 2, 50, 99, 100, 101}

func BenchmarkObserve(b *testing.B) {
	b.ReportAllocs()
	for i := range b.N {
		viewSink = cove.Observe(i, i+1)
	}
}

func BenchmarkNeed(b *testing.B) {
	req := cove.Req[int](func(v int) bool { return v > 0 })
	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		boolSink = cove.Need(requirementInputs[i%len(requirementInputs)], req)
	}
}

func BenchmarkNeedAll2(b *testing.B) {
	req := cove.All(
		func(v int) bool { return v > 0 },
		func(v int) bool { return v%2 == 0 },
	)
	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		boolSink = cove.Need(requirementInputs[i%len(requirementInputs)], req)
	}
}

func BenchmarkNeedAll3(b *testing.B) {
	req := cove.All(
		func(v int) bool { return v > 0 },
		func(v int) bool { return v%2 == 0 },
		func(v int) bool { return v < 100 },
	)
	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		boolSink = cove.Need(requirementInputs[i%len(requirementInputs)], req)
	}
}

func BenchmarkNeedExprAtom(b *testing.B) {
	req := cove.ExprAtom(func(v int) bool { return v > 0 })
	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		boolSink = cove.NeedExpr(requirementInputs[i%len(requirementInputs)], req)
	}
}

func BenchmarkNeedExprAll2(b *testing.B) {
	req := cove.ExprAll(
		cove.ExprAtom(func(v int) bool { return v > 0 }),
		cove.ExprAtom(func(v int) bool { return v%2 == 0 }),
	)
	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		boolSink = cove.NeedExpr(requirementInputs[i%len(requirementInputs)], req)
	}
}

func BenchmarkNeedExprAll3(b *testing.B) {
	req := cove.ExprAll(
		cove.ExprAtom(func(v int) bool { return v > 0 }),
		cove.ExprAtom(func(v int) bool { return v%2 == 0 }),
		cove.ExprAtom(func(v int) bool { return v < 100 }),
	)
	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		boolSink = cove.NeedExpr(requirementInputs[i%len(requirementInputs)], req)
	}
}

func BenchmarkReflectReqEval(b *testing.B) {
	req := cove.ReflectReq(cove.ExprAll(
		cove.ExprAtom(func(v int) bool { return v > 0 }),
		cove.ExprAtom(func(v int) bool { return v%2 == 0 }),
	))
	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		boolSink = cove.Need(requirementInputs[i%len(requirementInputs)], req)
	}
}

func BenchmarkReifyReqEval(b *testing.B) {
	req := cove.ReifyReq(cove.Req[int](func(v int) bool { return v > 0 && v%2 == 0 }))
	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		boolSink = cove.NeedExpr(requirementInputs[i%len(requirementInputs)], req)
	}
}

func BenchmarkStepExprWith(b *testing.B) {
	type ping struct{ kont.Phantom[int] }
	expr := kont.ExprBind(kont.ExprPerform(ping{}), func(v int) kont.Expr[int] {
		return kont.ExprReturn(v + 1)
	})
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		_, sv := cove.StepExprWith("ctx", expr)
		result, _ := sv.Resume(41)
		intSink = result
	}
}

func BenchmarkCheckRuleExpr(b *testing.B) {
	rule := cove.RequireExpr("positive", cove.ExprAtom(func(v int) bool { return v > 0 }))
	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		boolSink = cove.CheckRuleExpr(requirementInputs[i%len(requirementInputs)], rule).OK()
	}
}

func BenchmarkCheckedExprIntoView(b *testing.B) {
	checked := cove.GuardExpr(
		cove.ExprAtom(func(v int) bool { return v > 0 }),
		42,
	)
	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		viewSink, boolSink = checked.IntoView(requirementInputs[i%len(requirementInputs)])
	}
}

func BenchmarkResumeWith(b *testing.B) {
	type ping struct{ kont.Phantom[int] }
	expr := kont.ExprBind(kont.ExprPerform(ping{}), func(v int) kont.Expr[int] {
		return kont.ExprReturn(v + 1)
	})
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		_, sv := cove.StepExprWith(42, expr)
		result, done := sv.ResumeWith(41, func(v int) int { return v - 1 })
		viewSink = cove.View[int, int]{Ctx: done.Ask(), Value: result}
	}
}
