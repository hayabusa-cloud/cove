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

func BenchmarkObserve(b *testing.B) {
	for range b.N {
		_ = cove.Observe(1, 2)
	}
}

func BenchmarkNeed(b *testing.B) {
	req := cove.Req[int](func(v int) bool { return v > 0 })
	for range b.N {
		boolSink = cove.Need(1, req)
	}
}

func BenchmarkNeedAll2(b *testing.B) {
	req := cove.All(
		func(v int) bool { return v > 0 },
		func(v int) bool { return v%2 == 0 },
	)
	for range b.N {
		boolSink = cove.Need(2, req)
	}
}

func BenchmarkNeedAll3(b *testing.B) {
	req := cove.All(
		func(v int) bool { return v > 0 },
		func(v int) bool { return v%2 == 0 },
		func(v int) bool { return v < 100 },
	)
	for range b.N {
		boolSink = cove.Need(2, req)
	}
}

func BenchmarkNeedExprAtom(b *testing.B) {
	req := cove.ExprAtom(func(v int) bool { return v > 0 })
	for range b.N {
		boolSink = cove.NeedExpr(1, req)
	}
}

func BenchmarkNeedExprAll2(b *testing.B) {
	req := cove.ExprAll(
		cove.ExprAtom(func(v int) bool { return v > 0 }),
		cove.ExprAtom(func(v int) bool { return v%2 == 0 }),
	)
	for range b.N {
		boolSink = cove.NeedExpr(2, req)
	}
}

func BenchmarkNeedExprAll3(b *testing.B) {
	req := cove.ExprAll(
		cove.ExprAtom(func(v int) bool { return v > 0 }),
		cove.ExprAtom(func(v int) bool { return v%2 == 0 }),
		cove.ExprAtom(func(v int) bool { return v < 100 }),
	)
	for range b.N {
		boolSink = cove.NeedExpr(2, req)
	}
}

func BenchmarkReflectReqEval(b *testing.B) {
	req := cove.ReflectReq(cove.ExprAll(
		cove.ExprAtom(func(v int) bool { return v > 0 }),
		cove.ExprAtom(func(v int) bool { return v%2 == 0 }),
	))
	for range b.N {
		boolSink = cove.Need(2, req)
	}
}

func BenchmarkReifyReqEval(b *testing.B) {
	req := cove.ReifyReq(cove.Req[int](func(v int) bool { return v > 0 && v%2 == 0 }))
	for range b.N {
		boolSink = cove.NeedExpr(2, req)
	}
}

func BenchmarkStepExprWith(b *testing.B) {
	type ping struct{ kont.Phantom[int] }
	expr := kont.ExprBind(kont.ExprPerform(ping{}), func(v int) kont.Expr[int] {
		return kont.ExprReturn(v + 1)
	})
	for range b.N {
		_, sv := cove.StepExprWith("ctx", expr)
		sv.Resume(41)
	}
}

func BenchmarkCheckRuleExpr(b *testing.B) {
	rule := cove.RequireExpr("positive", cove.ExprAtom(func(v int) bool { return v > 0 }))
	for range b.N {
		boolSink = cove.CheckRuleExpr(2, rule).OK()
	}
}

func BenchmarkCheckedExprIntoView(b *testing.B) {
	checked := cove.GuardExpr(
		cove.ExprAtom(func(v int) bool { return v > 0 }),
		42,
	)
	for range b.N {
		_, boolSink = checked.IntoView(1)
	}
}

func BenchmarkResumeWith(b *testing.B) {
	type ping struct{ kont.Phantom[int] }
	expr := kont.ExprBind(kont.ExprPerform(ping{}), func(v int) kont.Expr[int] {
		return kont.ExprReturn(v + 1)
	})
	for range b.N {
		_, sv := cove.StepExprWith(42, expr)
		sv.ResumeWith(41, func(v int) int { return v - 1 })
	}
}
