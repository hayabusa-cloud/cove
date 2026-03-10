// ©Hayabusa Cloud Co., Ltd. 2026. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package cove_test

import (
	"fmt"

	"code.hybscloud.com/cove"
	"code.hybscloud.com/kont"
)

func ExampleExtend() {
	type ctx struct{ Budget int }
	v := cove.Observe(ctx{Budget: 4}, 10)
	next := cove.Extend(v, func(v cove.View[ctx, int]) int {
		return v.Value + v.Ctx.Budget
	})
	fmt.Println(next.Extract())
	// Output: 14
}

func ExampleAll() {
	type caps struct {
		CanSubmit bool
		HasToken  bool
	}
	req := cove.All(
		func(c caps) bool { return c.CanSubmit },
		func(c caps) bool { return c.HasToken },
	)
	fmt.Println(cove.Need(caps{CanSubmit: true, HasToken: true}, req))
	// Output: true
}

func ExamplePullback() {
	type runtime struct{ Budget int }
	req := cove.Pullback(func(budget int) bool { return budget > 0 }, func(rt runtime) int {
		return rt.Budget
	})
	fmt.Println(cove.Need(runtime{Budget: 2}, req))
	// Output: true
}

func ExampleReifyReq() {
	req := cove.Req[int](func(v int) bool { return v > 0 })
	expr := cove.ReifyReq(req)
	fmt.Println(cove.NeedExpr(1, expr))
	fmt.Println(cove.NeedExpr(-1, expr))
	// Output:
	// true
	// false
}

func ExampleReflectReq() {
	type runtime struct {
		Budget      int
		CanDispatch bool
	}
	expr := cove.ExprAll(
		cove.ExprAtom(func(r runtime) bool { return r.Budget > 0 }),
		cove.ExprAtom(func(r runtime) bool { return r.CanDispatch }),
	)
	req := cove.ReflectReq(expr)
	ctx := runtime{Budget: 2, CanDispatch: true}
	fmt.Println(cove.NeedExpr(ctx, expr))
	fmt.Println(cove.Need(ctx, req))
	// Output:
	// true
	// true
}

func ExampleStepExprWith() {
	type runtime struct{ Budget int }
	type ping struct{ kont.Phantom[int] }

	expr := kont.ExprBind(kont.ExprPerform(ping{}), func(v int) kont.Expr[int] {
		return kont.ExprReturn(v + 1)
	})

	_, step := cove.StepExprWith(runtime{Budget: 2}, expr)
	fmt.Println(step.Suspension != nil)
	fmt.Println(step.Ask().Budget)
	_, isPing := step.Op().(ping)
	fmt.Println(isPing)
	result, sv := step.Resume(41)
	fmt.Println(result, sv.Suspension != nil)
	// Output:
	// true
	// 2
	// true
	// 42 false
}

func ExampleGuardExpr() {
	type runtime struct{ Budget int }
	checked := cove.GuardExpr(
		cove.ExprAtom(func(r runtime) bool { return r.Budget > 0 }),
		"payload",
	)
	view, ok := checked.IntoView(runtime{Budget: 2})
	fmt.Println(ok, view.Extract())
	_, ok = checked.IntoView(runtime{Budget: 0})
	fmt.Println(ok)
	// Output:
	// true payload
	// false
}

func ExampleSuspensionView_ResumeWith() {
	type runtime struct {
		Budget int
		kont.Phantom[int]
	}
	type ping struct{ kont.Phantom[int] }

	expr := kont.ExprBind(kont.ExprPerform(ping{}), func(v int) kont.Expr[int] {
		return kont.ExprReturn(v + 1)
	})

	_, step := cove.StepExprWith(runtime{Budget: 10}, expr)
	fmt.Println(step.Ask().Budget)
	result, sv := step.ResumeWith(41, func(r runtime) runtime {
		r.Budget--
		return r
	})
	fmt.Println(result, sv.Suspension != nil)
	// Output:
	// 10
	// 42 false
}

func ExampleCheckRuleExpr() {
	type runtime struct{ Budget int }
	rule := cove.RequireExpr("budget", cove.ExprAtom(func(r runtime) bool {
		return r.Budget > 0
	}))
	report := cove.CheckRuleExpr(runtime{Budget: 5}, rule)
	fmt.Println(report.OK())
	report = cove.CheckRuleExpr(runtime{Budget: 0}, rule)
	fmt.Println(report.OK(), report.Failed)
	// Output:
	// true
	// false budget
}
