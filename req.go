// ©Hayabusa Cloud Co., Ltd. 2026. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package cove

// Req is a requirement over ambient context C.
type Req[C Ambient] func(C) bool

func trueReq[C Ambient](C) bool { return true }

func falseReq[C Ambient](C) bool { return false }

// Need reports whether ctx satisfies req.
// A nil requirement is treated as true.
func Need[C Ambient](ctx C, req Req[C]) bool {
	if req == nil {
		return true
	}
	return req(ctx)
}

// Pullback transports req along a context projection.
func Pullback[C, D Ambient](req Req[D], f func(C) D) Req[C] {
	if req == nil {
		return nil
	}
	return func(ctx C) bool {
		return req(f(ctx))
	}
}

// All returns the conjunction of reqs.
func All[C Ambient](reqs ...Req[C]) Req[C] {
	switch len(reqs) {
	case 0:
		return True[C]()
	case 1:
		return reqs[0]
	}
	return func(ctx C) bool {
		for _, req := range reqs {
			if !Need(ctx, req) {
				return false
			}
		}
		return true
	}
}

// Any returns the disjunction of reqs.
func Any[C Ambient](reqs ...Req[C]) Req[C] {
	switch len(reqs) {
	case 0:
		return False[C]()
	case 1:
		return reqs[0]
	}
	return func(ctx C) bool {
		for _, req := range reqs {
			if Need(ctx, req) {
				return true
			}
		}
		return false
	}
}

// Not returns the negation of req.
func Not[C Ambient](req Req[C]) Req[C] {
	if req == nil {
		return False[C]()
	}
	return func(ctx C) bool { return !req(ctx) }
}

// True returns a requirement that always holds.
func True[C Ambient]() Req[C] {
	return trueReq[C]
}

// False returns a requirement that never holds.
func False[C Ambient]() Req[C] {
	return falseReq[C]
}
