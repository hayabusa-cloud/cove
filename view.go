// ©Hayabusa Cloud Co., Ltd. 2026. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package cove

// View is a value under ambient context.
type View[C, A any] struct {
	Ctx   C
	Value A
}

// Observe constructs a contextual view.
func Observe[C, A any](ctx C, value A) View[C, A] {
	return View[C, A]{Ctx: ctx, Value: value}
}

// Extract returns the focused value.
func (v View[C, A]) Extract() A { return v.Value }

// Ask returns the ambient context.
func (v View[C, A]) Ask() C { return v.Ctx }

// MapContext transforms the ambient context and preserves the focused value.
func MapContext[C, D, A any](v View[C, A], f func(C) D) View[D, A] {
	return View[D, A]{Ctx: f(v.Ctx), Value: v.Value}
}

// Duplicate returns the current view as the focused value.
func Duplicate[C, A any](v View[C, A]) View[C, View[C, A]] {
	return View[C, View[C, A]]{Ctx: v.Ctx, Value: v}
}

// Extend maps the whole view to a new focused value.
func Extend[C, A, B any](v View[C, A], f func(View[C, A]) B) View[C, B] {
	return View[C, B]{Ctx: v.Ctx, Value: f(v)}
}

// Map transforms the focused value and preserves the ambient context.
func Map[C, A, B any](v View[C, A], f func(A) B) View[C, B] {
	return View[C, B]{Ctx: v.Ctx, Value: f(v.Value)}
}

// Replace substitutes the focused value and preserves the ambient context.
func Replace[C, A, B any](v View[C, A], value B) View[C, B] {
	return View[C, B]{Ctx: v.Ctx, Value: value}
}

// WithContext substitutes the ambient context and preserves the focused value.
func WithContext[C, A any](v View[C, A], ctx C) View[C, A] {
	v.Ctx = ctx
	return v
}
