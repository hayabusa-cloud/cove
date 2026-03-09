// ©Hayabusa Cloud Co., Ltd. 2026. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package cove

func intoViewIf[C Ambient, A Focus](ctx C, value A, req Req[C]) (View[C, A], bool) {
	if !Need(ctx, req) {
		var zero View[C, A]
		return zero, false
	}
	return Observe(ctx, value), true
}

// Checked couples a value with a requirement.
type Checked[C Ambient, A Focus] struct {
	Req   Req[C]
	Value A
}

// Guarded couples a value with a named rule.
type Guarded[C Ambient, A Focus] struct {
	Rule  Rule[C]
	Value A
}

// Guard constructs a Checked value.
func Guard[C Ambient, A Focus](req Req[C], value A) Checked[C, A] {
	return Checked[C, A]{Req: req, Value: value}
}

// Check reports whether ctx satisfies the requirement.
func (c Checked[C, A]) Check(ctx C) bool { return Need(ctx, c.Req) }

// PullbackChecked transports a checked value along a context projection.
func PullbackChecked[C, D Ambient, A Focus](checked Checked[C, A], f func(D) C) Checked[D, A] {
	return Checked[D, A]{Req: Pullback(checked.Req, f), Value: checked.Value}
}

// IntoView returns a view when the requirement holds.
func (c Checked[C, A]) IntoView(ctx C) (View[C, A], bool) {
	return intoViewIf(ctx, c.Value, c.Req)
}

// MustView panics if the requirement does not hold.
func (c Checked[C, A]) MustView(ctx C) View[C, A] {
	if v, ok := c.IntoView(ctx); ok {
		return v
	}
	panic("cove: requirement does not hold")
}

// MapChecked transforms the value and preserves its requirement.
func MapChecked[C Ambient, A, B Focus](c Checked[C, A], f func(A) B) Checked[C, B] {
	return Checked[C, B]{Req: c.Req, Value: f(c.Value)}
}

// GuardRule constructs a Guarded value.
func GuardRule[C Ambient, A Focus](rule Rule[C], value A) Guarded[C, A] {
	return Guarded[C, A]{Rule: rule, Value: value}
}

// Check evaluates the guarded rule.
func (g Guarded[C, A]) Check(ctx C) Report { return CheckRule(ctx, g.Rule) }

// PullbackGuarded transports a guarded value along a context projection.
func PullbackGuarded[C, D Ambient, A Focus](guarded Guarded[C, A], f func(D) C) Guarded[D, A] {
	return Guarded[D, A]{Rule: PullbackRule(guarded.Rule, f), Value: guarded.Value}
}

// IntoView returns a view and the rule report.
func (g Guarded[C, A]) IntoView(ctx C) (View[C, A], Report) {
	report := CheckRule(ctx, g.Rule)
	if !report.OK() {
		var zero View[C, A]
		return zero, report
	}
	return View[C, A]{Ctx: ctx, Value: g.Value}, report
}

// MapGuarded transforms the value and preserves the rule.
func MapGuarded[C Ambient, A, B Focus](g Guarded[C, A], f func(A) B) Guarded[C, B] {
	return Guarded[C, B]{Rule: g.Rule, Value: f(g.Value)}
}
