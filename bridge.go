// ©Hayabusa Cloud Co., Ltd. 2026. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package cove

import "code.hybscloud.com/kont"

// Reify bridges a Cont-world computation into the Expr world.
func Reify[A Focus](m kont.Cont[Resumed, A]) kont.Expr[A] {
	return kont.Reify(m)
}

// Reflect bridges an Expr-world computation into the Cont world.
func Reflect[A Focus](m kont.Expr[A]) kont.Cont[Resumed, A] {
	return kont.Reflect(m)
}

// ReifyReq wraps a [Req] as an [ExprAtom].
func ReifyReq[C Ambient](req Req[C]) ReqExpr[C] {
	return ExprAtom[C](req)
}

// ReflectReq evaluates an Expr-world requirement through [NeedExpr].
func ReflectReq[C Ambient](expr ReqExpr[C]) Req[C] {
	return func(ctx C) bool { return NeedExpr(ctx, expr) }
}
