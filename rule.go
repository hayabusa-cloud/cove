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

// Report is the result of checking one or more named rules.
type Report struct {
	Failed  RuleError // zero value means success
	Checked int       // number of rules examined before success or first failure
}

// OK reports whether all checked rules passed.
func (r Report) OK() bool { return r.Failed == "" }

// Require constructs a named rule.
func Require[C Ambient](name string, check Req[C]) Rule[C] {
	return Rule[C]{Name: name, Check: check}
}

// Req returns the underlying requirement.
func (r Rule[C]) Req() Req[C] { return r.Check }

// Match reports whether ctx satisfies the rule.
func (r Rule[C]) Match(ctx C) bool { return Need(ctx, r.Check) }

// PullbackRule transports a named rule along a context projection.
func PullbackRule[C, D Ambient](rule Rule[C], f func(D) C) Rule[D] {
	return Rule[D]{Name: rule.Name, Check: Pullback(rule.Check, f)}
}

// CheckRule evaluates a single rule.
func CheckRule[C Ambient](ctx C, rule Rule[C]) Report {
	if Need(ctx, rule.Check) {
		return Report{Checked: 1}
	}
	return Report{Failed: RuleError(rule.Name), Checked: 1}
}

// CheckRules evaluates rules left to right and stops at the first failure.
func CheckRules[C Ambient](ctx C, rules ...Rule[C]) Report {
	for i, rule := range rules {
		if !Need(ctx, rule.Check) {
			return Report{Failed: RuleError(rule.Name), Checked: i + 1}
		}
	}
	return Report{Checked: len(rules)}
}
