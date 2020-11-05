// Copyright (c) 2014-2016 The btcsuite developers
// Copyright (c) 2019 Caleb James DeLisle
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package blockchain

import (
	"fmt"

	"github.com/pkt-cash/pktd/btcutil/er"
)

// DeploymentError identifies an error that indicates a deployment ID was
// specified that does not exist.
type deploymentError0 uint32

// Error returns the assertion error as a human-readable string and satisfies
// the error interface.
func (e deploymentError0) Error() string {
	return fmt.Sprintf("deployment ID %d does not exist", uint32(e))
}

// DeploymentError ...
func DeploymentError(i uint32) er.R {
	return er.E(deploymentError0(i))
}

// AssertError identifies an error that indicates an internal code consistency
// issue and should be treated as a critical and unrecoverable error.
func AssertError(s string) er.R {
	return er.New("assertion failed: " + s)
}
