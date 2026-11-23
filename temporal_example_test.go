// ©Hayabusa Cloud Co., Ltd. 2026. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package cove_test

import (
	"fmt"

	"code.hybscloud.com/cove"
	"code.hybscloud.com/kont"
	"code.hybscloud.com/tempo"
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

// temporalValue is a pure synthetic operation; it has no external resource.
type temporalValue struct {
	kont.Phantom[int]
	input int
}

func ExampleCheckSuspension_temporal() {
	const generation = 7
	state := tempo.Both(temporalIdle, temporalEvidence{}).Step(temporalArm)
	// The calculation can run when this generation is ready and its prerequisite has completed.
	canRun := func(obs tempo.Pair[bool, temporalEvidence]) bool { return obs.Left && obs.Right.done }
	_, susp := cove.Step(kont.Perform(temporalValue{input: 41}))
	_, allowed := cove.CheckSuspension(state.Observe(), susp, canRun)
	fmt.Println("allowed after arming:", allowed)

	events := []struct {
		generation uint64
		kind       temporalEvent
	}{
		{6, temporalComplete},
		{generation, temporalReady},
		{generation, temporalComplete},
	}
	// The owner processes events and resumes the suspension serially.
	for _, event := range events {
		if event.generation != generation {
			fmt.Println("ignored event for generation", event.generation)
			continue
		}
		state = state.Step(event.kind)
		obs := state.Observe()
		checked, ok := cove.CheckSuspension(obs, susp, canRun)
		if !ok {
			fmt.Println("ready:", obs.Left, "prerequisite complete:", obs.Right.done)
			continue
		}
		op := checked.Op().(temporalValue)
		result, next := checked.Resume(op.input + 1)
		fmt.Println("result:", result, "completed:", next.Suspension == nil)
		return
	}
	susp.Discard()
	// Output:
	// allowed after arming: false
	// ignored event for generation 6
	// ready: true prerequisite complete: false
	// result: 42 completed: true
}
