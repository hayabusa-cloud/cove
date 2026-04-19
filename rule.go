// ©Hayabusa Cloud Co., Ltd. 2026. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package cove

// Rule names a requirement for diagnostics.
type Rule[C Ambient] struct {
	Name  string
	Check Req[C]
}

// RuleError names the first failed rule in a report.
type RuleError string

// Error returns the failed rule name.
func (e RuleError) Error() string {
	return string(e)
}

const unnamedRuleFailure RuleError = "cove: unnamed rule"

// Report records the result of checking one or more named rules.
type Report struct {
	Failed  RuleError // zero value means success
	Checked int       // number of rules examined before success or first failure
}

// OK reports whether every checked rule passed.
func (r Report) OK() bool { return r.Failed == "" }

// FailedRule reports the first failed rule name, or "" on success.
func (r Report) FailedRule() string { return string(r.Failed) }

// Err returns the first failed rule as an error, or nil on success.
func (r Report) Err() error {
	if r.OK() {
		return nil
	}
	return r.Failed
}

func requireRuleName(name string) {
	if name == "" {
		panic("cove: empty rule name")
	}
}

// ruleFailure converts a rule name to a failure label while keeping [Report.OK]
// sound for direct [Rule] literals that bypass [Require] and [RequireExpr].
func ruleFailure(name string) RuleError {
	if name == "" {
		return unnamedRuleFailure
	}
	return RuleError(name)
}

// Require constructs a named rule.
func Require[C Ambient](name string, check Req[C]) Rule[C] {
	requireRuleName(name)
	return Rule[C]{Name: name, Check: check}
}

// Req returns the underlying requirement.
func (r Rule[C]) Req() Req[C] { return r.Check }

// Match reports whether ctx satisfies r.
func (r Rule[C]) Match(ctx C) bool { return Need(ctx, r.Check) }

// PullbackRule transports r along a context projection.
func PullbackRule[C, D Ambient](rule Rule[C], f func(D) C) Rule[D] {
	return Rule[D]{Name: rule.Name, Check: Pullback(rule.Check, f)}
}

// CheckRule checks a single rule.
func CheckRule[C Ambient](ctx C, rule Rule[C]) Report {
	if Need(ctx, rule.Check) {
		return Report{Checked: 1}
	}
	return Report{Failed: ruleFailure(rule.Name), Checked: 1}
}

// CheckRules checks rules from left to right and stops at the first failure.
func CheckRules[C Ambient](ctx C, rules ...Rule[C]) Report {
	for i, rule := range rules {
		if !Need(ctx, rule.Check) {
			return Report{Failed: ruleFailure(rule.Name), Checked: i + 1}
		}
	}
	return Report{Checked: len(rules)}
}
