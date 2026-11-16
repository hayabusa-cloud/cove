// ©Hayabusa Cloud Co., Ltd. 2026. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package cove_test

import (
	"fmt"

	"code.hybscloud.com/cove"
	"code.hybscloud.com/kont"
)

type temporalEvent uint8

const (
	temporalArm temporalEvent = iota
	temporalReady
	temporalExpire
	temporalCancel
	temporalComplete
	temporalReturned
	temporalOther
)

type temporalLife uint8

const (
	temporalIdle temporalLife = iota
	temporalArmed
	temporalLive
	temporalStopped
)

func (s temporalLife) Observe() bool { return s == temporalLive }

func (s temporalLife) Step(e temporalEvent) temporalLife {
	if s == temporalStopped {
		return s
	}
	switch e {
	case temporalExpire, temporalCancel:
		return temporalStopped
	case temporalArm:
		if s == temporalIdle {
			return temporalArmed
		}
	case temporalReady:
		if s == temporalArmed {
			return temporalLive
		}
	}
	return s
}

type temporalEvidence struct{ done, returned bool }

func (s temporalEvidence) Observe() temporalEvidence { return s }
func (s temporalEvidence) Step(e temporalEvent) temporalEvidence {
	switch e {
	case temporalComplete:
		s.done = true
	case temporalReturned:
		s.returned = true
	}
	return s
}

type temporalContext struct {
	eligible   bool
	generation uint64
}

func temporalAllowed(c temporalContext) bool { return c.eligible }

// temporalValue is a pure synthetic operation; it has no external resource.
type temporalValue struct {
	kont.Phantom[int]
	input int
}

func ExampleCheckSuspension_temporal() {
	life := temporalIdle.Step(temporalArm)
	ctx := temporalContext{eligible: life.Observe(), generation: 7}
	_, owner := cove.StepWith(ctx, kont.Perform(temporalValue{input: 41}))
	_, allowed := cove.CheckSuspension(owner.Ctx, owner.Suspension, temporalAllowed)
	fmt.Println("armed:", allowed)
	// Refusal leaves owner intact. The owner correlates the new fact, refreshes
	// context and protects this local check/resume sequence from invalidation.
	life = life.Step(temporalReady)
	ctx.eligible = life.Observe()
	owner = cove.WithContextSuspension(owner, ctx)
	checked, allowed := cove.CheckSuspension(owner.Ctx, owner.Suspension, temporalAllowed)
	if !allowed {
		owner.Discard()
		return
	}
	op := checked.Op().(temporalValue)
	result, next := checked.Resume(op.input + 1)
	fmt.Println("completed:", result, next.Suspension == nil)
	// Output:
	// armed: false
	// completed: 42 true
}
