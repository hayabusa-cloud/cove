// ©Hayabusa Cloud Co., Ltd. 2026. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package cove

// RuleExpr names an Expr-world requirement for diagnostics.
type RuleExpr[C Ambient] struct {
	Name  string
	Check ReqExpr[C]
}

// RequireExpr constructs a named Expr-world rule.
func RequireExpr[C Ambient](name string, check ReqExpr[C]) RuleExpr[C] {
	return RuleExpr[C]{Name: name, Check: check}
}

// Req returns the underlying Expr-world requirement.
func (r RuleExpr[C]) Req() ReqExpr[C] { return r.Check }

// Match reports whether ctx satisfies the rule.
func (r RuleExpr[C]) Match(ctx C) bool { return NeedExpr(ctx, r.Check) }

// PullbackRuleExpr transports a rule along a context projection.
func PullbackRuleExpr[C, D Ambient](rule RuleExpr[C], f func(D) C) RuleExpr[D] {
	return RuleExpr[D]{Name: rule.Name, Check: ExprPullback(rule.Check, f)}
}

// CheckRuleExpr evaluates a single rule.
func CheckRuleExpr[C Ambient](ctx C, rule RuleExpr[C]) Report {
	if NeedExpr(ctx, rule.Check) {
		return Report{Checked: 1}
	}
	return Report{Failed: RuleError(rule.Name), Checked: 1}
}

// CheckRulesExpr evaluates rules left to right and stops at the first failure.
func CheckRulesExpr[C Ambient](ctx C, rules ...RuleExpr[C]) Report {
	for i, rule := range rules {
		if !NeedExpr(ctx, rule.Check) {
			return Report{Failed: RuleError(rule.Name), Checked: i + 1}
		}
	}
	return Report{Checked: len(rules)}
}
