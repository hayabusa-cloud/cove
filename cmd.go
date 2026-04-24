// ©Hayabusa Cloud Co., Ltd. 2026. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package cove

// Cmd is the coKleisli arrow for [View]: it consumes a contextual observation
// and produces a focus result without hiding the ambient context.
type Cmd[C Ambient, A, B Focus] func(View[C, A]) B

// Run applies cmd to v.
func Run[C Ambient, A, B Focus](v View[C, A], cmd Cmd[C, A, B]) B {
	return cmd(v)
}

// ExtractCmd returns the focused value and serves as the identity command for
// [View]: Compose(ExtractCmd, f) == f and Compose(g, ExtractCmd) == g.
func ExtractCmd[C Ambient, A Focus](v View[C, A]) A {
	return v.Extract()
}

// LiftCmd lifts a focus-only function into a contextual command.
func LiftCmd[C Ambient, A, B Focus](f func(A) B) Cmd[C, A, B] {
	return func(v View[C, A]) B {
		return f(v.Extract())
	}
}

// Compose composes contextual commands through [Extend], satisfying the
// coKleisli composition law Compose(g, f)(v) == g(Extend(v, f)).
func Compose[C Ambient, A, B, D Focus](g Cmd[C, B, D], f Cmd[C, A, B]) Cmd[C, A, D] {
	return func(v View[C, A]) D {
		return g(Extend(v, f))
	}
}
