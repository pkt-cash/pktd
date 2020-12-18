// Copyright (c) 2019 Caleb James DeLisle
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package util

// parentBlockHeight -> x -> y -> valid
//         0            1    2      3
const Conf_PacketCrypt_ANN_WAIT_PERIOD = 3

// How many times to run the random hash program per announcement hash cycle
const Conf_AnnHash_RANDHASH_CYCLES = 4

// How many times to run memohash when creating an item for announcement hashing
const Conf_AnnHash_MEMOHASH_CYCLES = 2

// ============================================================================
// RandHash / RandGen parameters
// ============================================================================

// RandHash rules, if a program requires fewer than MIN_OPS
// or more than MAX_OPS cycles to complete, it will be deemed invalid
// and the hashing attempt is failed.
const (
	Conf_RandHash_MIN_OPS = 0
	Conf_RandHash_MAX_OPS = 20000
)

// How complex of a program do we want to create ?
// Most operations have a "cost" equal to the number of inputs to the
// op. Loops multiply the cost of the operations within by the number
// of cycles of the loop. RandGen stops generating the hash program when
// the budget is exhausted.
const Conf_RandGen_INITIAL_BUDGET = 20000

// Programs which are created with fewer than MIN_INSNS or more than
// MAX_INSNS are deemed invalid and the hash attempt is failed.
const (
	Conf_RandGen_MIN_INSNS = 0
	Conf_RandGen_MAX_INSNS = 2048
)

// Some operations are more complicated than normal and have particular costs.
const (
	Conf_RandGen_MEMORY_COST = 20
	Conf_RandGen_INPUT_COST  = 2
	Conf_RandGen_BRANCH_COST = 50
)

// Loops have random numbers of cycles, these are the bounds of the random
// numbers. The max cycles become more as the scope depth grows, this helps
// stabilize the size of programs because outer loops have few cycles and thus
// spend less budget but inner loops have many more.
const Conf_RandGen_LOOP_MIN_CYCLES = 2

func Conf_RandGen_LOOP_MAX_CYCLES(scopeDepth int) int {
	return (7 + scopeDepth*29)
}

// Likelihood in a scope that a loop or branch will be created.
// Loops have a flat 23 in 32 chance while branches become less likely as the
// number of inctructions already emitted approaches the maximum. This helps
// to stabilize the size of generated programs.
func Conf_RandGen_SHOULD_LOOP(rand uint32) bool {
	return (((rand) % 32) < 23)
}

func Conf_RandGen_SHOULD_BRANCH(rand uint32, insnCount int) bool {
	return ((int((rand)%64) + (insnCount * 25 / Conf_RandGen_MAX_INSNS)) < 50)
}

// How much budget remains after we enter an if sub-scope
// Technically it should be 100% because only one of the two branches will be taken
// but reducing it a bit helps make the code more compact and convoluted.
func Conf_RandGen_IF_BODY_BUDGET(budget int, scopes int) int {
	return (((budget) * 7) / 32)
}

// 50% chance that an if statement is completely unpredictable
const Conf_RandGen_RANDOM_BRANCH_LIKELIHOOD = 2

// 25% chance that an input variable will come from a higher scope
const Conf_RandGen_HIGHER_SCOPE_LIKELIHOOD = 4

// 12.5% chance that a variable used in an op is one which has been used before
const Conf_RandGen_VAR_REUSE_LIKELIHOOD = 8

// 25% chance that an op uses an immediate input rather than a variable
const Conf_RandGen_IMMEDIATE_LIKELIHOOD = 4
