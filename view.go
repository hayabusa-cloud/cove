// ©Hayabusa Cloud Co., Ltd. 2026. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package cove

// View is a value in focus under ambient context.
type View[C Ambient, A Focus] struct {
	Ctx   C
	Value A
}

// Observe constructs a contextual view.
func Observe[C Ambient, A Focus](ctx C, value A) View[C, A] {
	return View[C, A]{Ctx: ctx, Value: value}
}

// Extract returns the value in focus.
func (v View[C, A]) Extract() A { return v.Value }

// Ask returns the ambient context.
func (v View[C, A]) Ask() C { return v.Ctx }

// MapContext transforms the ambient context and preserves the value in focus.
func MapContext[C, D Ambient, A Focus](v View[C, A], f func(C) D) View[D, A] {
	return View[D, A]{Ctx: f(v.Ctx), Value: v.Value}
}

// Duplicate returns the current view as the value in focus.
func Duplicate[C Ambient, A Focus](v View[C, A]) View[C, View[C, A]] {
	return View[C, View[C, A]]{Ctx: v.Ctx, Value: v}
}

// Extend maps the whole view to a new value in focus.
func Extend[C Ambient, A, B Focus](v View[C, A], f func(View[C, A]) B) View[C, B] {
	return View[C, B]{Ctx: v.Ctx, Value: f(v)}
}

// Map transforms the value in focus and preserves the ambient context.
func Map[C Ambient, A, B Focus](v View[C, A], f func(A) B) View[C, B] {
	return View[C, B]{Ctx: v.Ctx, Value: f(v.Value)}
}

// Replace substitutes the value in focus and preserves the ambient context.
func Replace[C Ambient, A, B Focus](v View[C, A], value B) View[C, B] {
	return View[C, B]{Ctx: v.Ctx, Value: value}
}

// WithContext substitutes the ambient context and preserves the value in focus.
func WithContext[C Ambient, A Focus](v View[C, A], ctx C) View[C, A] {
	v.Ctx = ctx
	return v
}
