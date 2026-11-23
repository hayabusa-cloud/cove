// ©Hayabusa Cloud Co., Ltd. 2026. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package cove

import "code.hybscloud.com/kont"

// Reify runs m to its first return or operation and captures the result as a [kont.Expr].
// A completed result is reusable. A pending result and its copies share one invocation.
// Stepping it a second time through any copy panics before yielding the operation.
// Reify a reusable computation again for each run. Use [StepWith] to step m directly.
func Reify[A Focus](m kont.Cont[Resumed, A]) kont.Expr[A] {
	return kont.Reify(m)
}

// Reflect converts m to a [kont.Cont]. If m is a pending [Reify] result,
// evaluating the returned computation consumes that same invocation.
// Use [StepExprWith] to step m directly.
func Reflect[A Focus](m kont.Expr[A]) kont.Cont[Resumed, A] {
	return kont.Reflect(m)
}

// ReifyReq wraps req as one [ExprAtom]. The Boolean structure of a [ReflectReq]
// result stays opaque. If req is nil, [NeedExpr] panics when it reaches the atom.
// Use [ExprTrue] for an unrestricted expression.
func ReifyReq[C Ambient](req Req[C]) ReqExpr[C] {
	return ExprAtom[C](req)
}

// ReflectReq returns a [Req] that evaluates expr with [NeedExpr].
func ReflectReq[C Ambient](expr ReqExpr[C]) Req[C] {
	return func(ctx C) bool { return NeedExpr(ctx, expr) }
}
