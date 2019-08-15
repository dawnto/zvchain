//   Copyright (C) 2018 ZVChain
//
//   This program is free software: you can redistribute it and/or modify
//   it under the terms of the GNU General Public License as published by
//   the Free Software Foundation, either version 3 of the License, or
//   (at your option) any later version.
//
//   This program is distributed in the hope that it will be useful,
//   but WITHOUT ANY WARRANTY; without even the implied warranty of
//   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
//   GNU General Public License for more details.
//
//   You should have received a copy of the GNU General Public License
//   along with this program.  If not, see <https://www.gnu.org/licenses/>.

package model

import (
	"fmt"
	"sync"

	"github.com/zvchain/zvchain/consensus/groupsig"
)

// GroupSignGenerator defines the group signature generator.
// It accepts signature pieces util the threshold is reached and then the group signature will be recovered
type GroupSignGenerator struct {
	witnesses map[string]groupsig.Signature // Signature pieces
	threshold int                           // Threshold
	gSign     groupsig.Signature            // Group signature generated
	lock      sync.RWMutex
}

func NewGroupSignGenerator(threshold int) *GroupSignGenerator {
	return &GroupSignGenerator{
		witnesses: make(map[string]groupsig.Signature, 0),
		threshold: threshold,
	}
}

func (gs *GroupSignGenerator) Threshold() int {
	return gs.threshold
}

func (gs *GroupSignGenerator) GetWitness(id groupsig.ID) (groupsig.Signature, bool) {
	gs.lock.RLock()
	defer gs.lock.RUnlock()
	if s, ok := gs.witnesses[id.GetAddrString()]; ok {
		return s, true
	}
	return groupsig.Signature{}, false
}

// AddWitnessForce do not check if it has been recovered, just add the piece
func (gs *GroupSignGenerator) AddWitnessForce(id groupsig.ID, signature groupsig.Signature) (add bool, generated bool) {
	gs.lock.Lock()
	defer gs.lock.Unlock()

	key := id.GetAddrString()
	if _, ok := gs.witnesses[key]; ok {
		return false, false
	}
	gs.witnesses[key] = signature

	if len(gs.witnesses) >= gs.threshold {
		return true, gs.generate()
	}
	return true, false
}

// AddWitness try to add the piece.
// It will ignore the piece after group signature is recovered
func (gs *GroupSignGenerator) AddWitness(id groupsig.ID, signature groupsig.Signature) (add bool, generated bool) {
	if gs.Recovered() {
		return false, true
	}

	return gs.AddWitnessForce(id, signature)
}

func (gs *GroupSignGenerator) generate() bool {
	if gs.gSign.IsValid() {
		return true
	}

	sig := groupsig.RecoverSignatureByMapI(gs.witnesses, gs.threshold)
	if sig == nil {
		return false
	}
	gs.gSign = *sig
	return true
}

func (gs *GroupSignGenerator) GetGroupSign() groupsig.Signature {
	gs.lock.RLock()
	defer gs.lock.RUnlock()

	return gs.gSign
}

// VerifyGroupSign verifies the signature generated by the generator
func (gs *GroupSignGenerator) VerifyGroupSign(gpk groupsig.Pubkey, data []byte) bool {
	return groupsig.VerifySig(gpk, data, gs.GetGroupSign())
}

func (gs *GroupSignGenerator) WitnessSize() int {
	gs.lock.RLock()
	defer gs.lock.RUnlock()
	return len(gs.witnesses)
}

func (gs *GroupSignGenerator) ThresholdReached() bool {
	return gs.WitnessSize() >= gs.threshold
}

func (gs *GroupSignGenerator) Recovered() bool {
	gs.lock.RLock()
	defer gs.lock.RUnlock()
	return gs.gSign.IsValid()
}

func (gs *GroupSignGenerator) ForEachWitness(f func(id string, sig groupsig.Signature) bool) {
	gs.lock.RLock()
	defer gs.lock.RUnlock()

	for ids, sig := range gs.witnesses {
		if !f(ids, sig) {
			break
		}
	}
}

func (gs *GroupSignGenerator) Brief() string {
	return fmt.Sprintf("current piece-size %v，threshold %v", gs.WitnessSize(), gs.threshold)
}
