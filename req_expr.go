// ©Hayabusa Cloud Co., Ltd. 2026. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package cove

type reqKind uint8

const (
	reqTrue reqKind = iota // zero value, no requirements
	reqFalse
	reqAtom
	reqNot
	reqAll
	reqAny
)

// ReqExpr stores a requirement as data instead of a closure.
// The zero value is [ExprTrue].
type ReqExpr[C Ambient] struct {
	kind reqKind
	fn   func(C) bool // reqAtom
	sub  []ReqExpr[C] // reqNot stores one child in sub[0], reqAll and reqAny store all children
}

// NeedExpr reports whether ctx satisfies req.
func NeedExpr[C Ambient](ctx C, req ReqExpr[C]) bool {
	switch req.kind {
	case reqFalse:
		return false
	case reqAtom:
		return req.fn(ctx)
	case reqNot:
		return !NeedExpr(ctx, req.sub[0])
	case reqAll:
		for i := range req.sub {
			if !NeedExpr(ctx, req.sub[i]) {
				return false
			}
		}
	case reqAny:
		for i := range req.sub {
			if NeedExpr(ctx, req.sub[i]) {
				return true
			}
		}
		return false
	}
	return true
}

// ExprTrue returns a requirement that always holds.
func ExprTrue[C Ambient]() ReqExpr[C] {
	return ReqExpr[C]{kind: reqTrue}
}

// ExprFalse returns a requirement that never holds.
func ExprFalse[C Ambient]() ReqExpr[C] {
	return ReqExpr[C]{kind: reqFalse}
}

// ExprAtom wraps a leaf predicate as a requirement expression.
func ExprAtom[C Ambient](fn func(C) bool) ReqExpr[C] {
	return ReqExpr[C]{kind: reqAtom, fn: fn}
}

// ExprNot returns the negation of inner.
func ExprNot[C Ambient](inner ReqExpr[C]) ReqExpr[C] {
	return ReqExpr[C]{kind: reqNot, sub: []ReqExpr[C]{inner}}
}

// ExprAll returns the conjunction of reqs.
func ExprAll[C Ambient](reqs ...ReqExpr[C]) ReqExpr[C] {
	switch len(reqs) {
	case 0:
		return ExprTrue[C]()
	case 1:
		return reqs[0]
	default:
		return ReqExpr[C]{kind: reqAll, sub: reqs}
	}
}

// ExprAny returns the disjunction of reqs.
func ExprAny[C Ambient](reqs ...ReqExpr[C]) ReqExpr[C] {
	switch len(reqs) {
	case 0:
		return ExprFalse[C]()
	case 1:
		return reqs[0]
	default:
		return ReqExpr[C]{kind: reqAny, sub: reqs}
	}
}

// ExprPullback transports req along a context projection and preserves its
// explicit Boolean structure.
func ExprPullback[C, D Ambient](req ReqExpr[D], f func(C) D) ReqExpr[C] {
	switch req.kind {
	case reqFalse:
		return ExprFalse[C]()
	case reqAtom:
		return ExprAtom(func(ctx C) bool { return req.fn(f(ctx)) })
	case reqNot:
		return ExprNot(ExprPullback(req.sub[0], f))
	case reqAll:
		sub := make([]ReqExpr[C], len(req.sub))
		for i := range req.sub {
			sub[i] = ExprPullback(req.sub[i], f)
		}
		return ExprAll(sub...)
	case reqAny:
		sub := make([]ReqExpr[C], len(req.sub))
		for i := range req.sub {
			sub[i] = ExprPullback(req.sub[i], f)
		}
		return ExprAny(sub...)
	}
	return ExprTrue[C]()
}
