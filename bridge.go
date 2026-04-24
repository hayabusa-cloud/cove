// ©Hayabusa Cloud Co., Ltd. 2026. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package cove

import "code.hybscloud.com/kont"

// Reify converts a [kont.Cont] computation to [kont.Expr].
// Prefer [StepWith] or [StepExprWith] when the caller needs the contextual
// suspension boundary.
func Reify[A Focus](m kont.Cont[Resumed, A]) kont.Expr[A] {
	return kont.Reify(m)
}

// Reflect converts a [kont.Expr] computation to [kont.Cont].
// Prefer [StepWith] or [StepExprWith] when the caller needs the contextual
// suspension boundary.
func Reflect[A Focus](m kont.Expr[A]) kont.Cont[Resumed, A] {
	return kont.Reflect(m)
}

// ReifyReq wraps a closure-form [Req] as an [ExprAtom].
// It is a lossy quotation helper for bridging into Expr-world composition, not a
// structural inverse of [ReflectReq]. In particular, ReifyReq(nil) is invalid
// and panics when the resulting expression is evaluated through [NeedExpr].
func ReifyReq[C Ambient](req Req[C]) ReqExpr[C] {
	return ExprAtom[C](req)
}

// ReflectReq returns a closure-form requirement that delegates to [NeedExpr].
// It preserves the observable predicate, but once converted back to [Req] the
// original Expr structure is no longer recoverable from the closure alone.
func ReflectReq[C Ambient](expr ReqExpr[C]) Req[C] {
	return func(ctx C) bool { return NeedExpr(ctx, expr) }
}
