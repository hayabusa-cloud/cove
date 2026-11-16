// ©Hayabusa Cloud Co., Ltd. 2026. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package cove_test

import (
	"testing"

	"code.hybscloud.com/cove"
)

func transitionOrdered(before, after int) bool { return before <= after }
func transitionEligible(world int) bool        { return world >= 2 }

var transitionSink bool

func BenchmarkForcingTransition(b *testing.B) {
	b.Run("Direct", func(b *testing.B) {
		b.ReportAllocs()
		for i := range b.N {
			before, after := i&3, (i>>2)&3
			transitionSink = transitionOrdered(before, after) && transitionEligible(before) && transitionEligible(after)
		}
	})
	b.Run("Cove", func(b *testing.B) {
		forcing := cove.Force[int](transitionOrdered, transitionEligible)
		b.ReportAllocs()
		b.ResetTimer()
		for i := range b.N {
			transitionSink = cove.CheckForcingTransition(forcing, cove.Transition[int]{Before: i & 3, After: (i >> 2) & 3})
		}
	})
}

// BenchmarkForcingTransitionPaths separates successful and refused checks.
func BenchmarkForcingTransitionPaths(b *testing.B) {
	for _, tc := range []struct {
		name                string
		beforeMask          int
		beforeAdd, afterAdd int
		req                 cove.Req[int]
	}{
		{"EdgeRejected", 7, 16, 0, transitionEligible},
		{"BeforeRejected", 1, 0, 8, transitionEligible},
		{"AfterRejected", 3, 2, 8, func(w int) bool { return w >= 2 && w < 8 }},
		{"Accepted", 7, 2, 16, transitionEligible},
		{"NilRequirement", 7, 0, 8, nil},
	} {
		b.Run(tc.name, func(b *testing.B) {
			forcing := cove.Force[int](transitionOrdered, tc.req)
			b.ReportAllocs()
			b.ResetTimer()
			for i := range b.N {
				transition := cove.Transition[int]{
					Before: (i & tc.beforeMask) + tc.beforeAdd,
					After:  ((i >> 3) & 7) + tc.afterAdd,
				}
				transitionSink = cove.CheckForcingTransition(forcing, transition)
			}
		})
	}
}
