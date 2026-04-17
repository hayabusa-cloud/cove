// ©Hayabusa Cloud Co., Ltd. 2026. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package cove

// Cmd is a contextual command over [View].
type Cmd[C Ambient, A, B Focus] func(View[C, A]) B

// Run applies a command to a contextual view.
func Run[C Ambient, A, B Focus](v View[C, A], cmd Cmd[C, A, B]) B {
	return cmd(v)
}

// ExtractCmd is the identity command for [View].
func ExtractCmd[C Ambient, A Focus](v View[C, A]) A {
	return v.Extract()
}

// LiftCmd turns a focus-only transform into a contextual command.
func LiftCmd[C Ambient, A, B Focus](f func(A) B) Cmd[C, A, B] {
	return func(v View[C, A]) B {
		return f(v.Extract())
	}
}

// Compose composes contextual commands through [Extend].
func Compose[C Ambient, A, B, D Focus](g Cmd[C, B, D], f Cmd[C, A, B]) Cmd[C, A, D] {
	return func(v View[C, A]) D {
		return g(Extend(v, f))
	}
}
