// ©Hayabusa Cloud Co., Ltd. 2026. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package cove

func intoViewIfExpr[C Ambient, A Focus](ctx C, value A, req ReqExpr[C]) (View[C, A], bool) {
	if !NeedExpr(ctx, req) {
		var zero View[C, A]
		return zero, false
	}
	return Observe(ctx, value), true
}

// CheckedExpr pairs a value with a requirement expression.
type CheckedExpr[C Ambient, A Focus] struct {
	Req   ReqExpr[C]
	Value A
}

// GuardedExpr pairs a value with a named requirement expression.
type GuardedExpr[C Ambient, A Focus] struct {
	Rule  RuleExpr[C]
	Value A
}

// GuardExpr constructs a checked value from a requirement expression.
func GuardExpr[C Ambient, A Focus](req ReqExpr[C], value A) CheckedExpr[C, A] {
	return CheckedExpr[C, A]{Req: req, Value: value}
}

// Check reports whether ctx satisfies c's requirement.
func (c CheckedExpr[C, A]) Check(ctx C) bool { return NeedExpr(ctx, c.Req) }

// PullbackCheckedExpr transports c along a context projection.
func PullbackCheckedExpr[C, D Ambient, A Focus](checked CheckedExpr[C, A], f func(D) C) CheckedExpr[D, A] {
	return CheckedExpr[D, A]{Req: ExprPullback(checked.Req, f), Value: checked.Value}
}

// IntoView returns a view when c's requirement holds.
func (c CheckedExpr[C, A]) IntoView(ctx C) (View[C, A], bool) {
	return intoViewIfExpr(ctx, c.Value, c.Req)
}

// MustView returns a view and panics if c's requirement does not hold.
func (c CheckedExpr[C, A]) MustView(ctx C) View[C, A] {
	if v, ok := c.IntoView(ctx); ok {
		return v
	}
	panic("cove: requirement does not hold")
}

// MapCheckedExpr maps the value and preserves the requirement.
func MapCheckedExpr[C Ambient, A, B Focus](c CheckedExpr[C, A], f func(A) B) CheckedExpr[C, B] {
	return CheckedExpr[C, B]{Req: c.Req, Value: f(c.Value)}
}

// GuardRuleExpr constructs a guarded value from a rule expression.
func GuardRuleExpr[C Ambient, A Focus](rule RuleExpr[C], value A) GuardedExpr[C, A] {
	return GuardedExpr[C, A]{Rule: rule, Value: value}
}

// Check checks the guarded rule against ctx.
func (g GuardedExpr[C, A]) Check(ctx C) Report { return CheckRuleExpr(ctx, g.Rule) }

// PullbackGuardedExpr transports g along a context projection.
func PullbackGuardedExpr[C, D Ambient, A Focus](guarded GuardedExpr[C, A], f func(D) C) GuardedExpr[D, A] {
	return GuardedExpr[D, A]{Rule: PullbackRuleExpr(guarded.Rule, f), Value: guarded.Value}
}

// IntoView returns a view together with the rule report.
func (g GuardedExpr[C, A]) IntoView(ctx C) (View[C, A], Report) {
	report := CheckRuleExpr(ctx, g.Rule)
	if !report.OK() {
		var zero View[C, A]
		return zero, report
	}
	return View[C, A]{Ctx: ctx, Value: g.Value}, report
}

// MapGuardedExpr maps the value and preserves the rule.
func MapGuardedExpr[C Ambient, A, B Focus](g GuardedExpr[C, A], f func(A) B) GuardedExpr[C, B] {
	return GuardedExpr[C, B]{Rule: g.Rule, Value: f(g.Value)}
}
