// Copyright 2020-2025 Consensys Software Inc.
// Licensed under the Apache License, Version 2.0. See the LICENSE file for details.

// Code generated by consensys/gnark-crypto DO NOT EDIT

package sumcheck

import (
	"fmt"
	fiatshamir "github.com/consensys/gnark-crypto/fiat-shamir"
	"github.com/consensys/gnark-crypto/internal/generator/test_vector_utils/small_rational"
	"github.com/consensys/gnark-crypto/internal/generator/test_vector_utils/small_rational/polynomial"
	"github.com/consensys/gnark-crypto/internal/generator/test_vector_utils/small_rational/test_vector_utils"
	"github.com/stretchr/testify/assert"
	"hash"
	"math/bits"
	"strings"
	"testing"
)

type singleMultilinClaim struct {
	g polynomial.MultiLin
}

func (c singleMultilinClaim) ProveFinalEval(r []small_rational.SmallRational) interface{} {
	return nil // verifier can compute the final eval itself
}

func (c singleMultilinClaim) VarsNum() int {
	return bits.TrailingZeros(uint(len(c.g)))
}

func (c singleMultilinClaim) ClaimsNum() int {
	return 1
}

func sumForX1One(g polynomial.MultiLin) polynomial.Polynomial {
	sum := g[len(g)/2]
	for i := len(g)/2 + 1; i < len(g); i++ {
		sum.Add(&sum, &g[i])
	}
	return []small_rational.SmallRational{sum}
}

func (c singleMultilinClaim) Combine(small_rational.SmallRational) polynomial.Polynomial {
	return sumForX1One(c.g)
}

func (c *singleMultilinClaim) Next(r small_rational.SmallRational) polynomial.Polynomial {
	c.g.Fold(r)
	return sumForX1One(c.g)
}

type singleMultilinLazyClaim struct {
	g          polynomial.MultiLin
	claimedSum small_rational.SmallRational
}

func (c singleMultilinLazyClaim) VerifyFinalEval(r []small_rational.SmallRational, combinationCoeff small_rational.SmallRational, purportedValue small_rational.SmallRational, proof interface{}) error {
	val := c.g.Evaluate(r, nil)
	if val.Equal(&purportedValue) {
		return nil
	}
	return fmt.Errorf("mismatch")
}

func (c singleMultilinLazyClaim) CombinedSum(combinationCoeffs small_rational.SmallRational) small_rational.SmallRational {
	return c.claimedSum
}

func (c singleMultilinLazyClaim) Degree(i int) int {
	return 1
}

func (c singleMultilinLazyClaim) ClaimsNum() int {
	return 1
}

func (c singleMultilinLazyClaim) VarsNum() int {
	return bits.TrailingZeros(uint(len(c.g)))
}

func testSumcheckSingleClaimMultilin(polyInt []uint64, hashGenerator func() hash.Hash) error {
	poly := make(polynomial.MultiLin, len(polyInt))
	for i, n := range polyInt {
		poly[i].SetUint64(n)
	}

	claim := singleMultilinClaim{g: poly.Clone()}

	proof, err := Prove(&claim, fiatshamir.WithHash(hashGenerator()))
	if err != nil {
		return err
	}

	var sb strings.Builder
	for _, p := range proof.PartialSumPolys {

		sb.WriteString("\t{")
		for i := 0; i < len(p); i++ {
			sb.WriteString(p[i].String())
			if i+1 < len(p) {
				sb.WriteString(", ")
			}
		}
		sb.WriteString("}\n")
	}

	lazyClaim := singleMultilinLazyClaim{g: poly, claimedSum: poly.Sum()}
	if err = Verify(lazyClaim, proof, fiatshamir.WithHash(hashGenerator())); err != nil {
		return err
	}

	proof.PartialSumPolys[0][0].Add(&proof.PartialSumPolys[0][0], test_vector_utils.ToElement(1))
	lazyClaim = singleMultilinLazyClaim{g: poly, claimedSum: poly.Sum()}
	if Verify(lazyClaim, proof, fiatshamir.WithHash(hashGenerator())) == nil {
		return fmt.Errorf("bad proof accepted")
	}
	return nil
}

func TestSumcheckDeterministicHashSingleClaimMultilin(t *testing.T) {
	//printMsws(36)

	polys := [][]uint64{
		{1, 2, 3, 4},             // 1 + 2X₁ + X₂
		{1, 2, 3, 4, 5, 6, 7, 8}, // 1 + 4X₁ + 2X₂ + X₃
		{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}, // 1 + 8X₁ + 4X₂ + 2X₃ + X₄
	}

	const MaxStep = 4
	const MaxStart = 4
	hashGens := make([]func() hash.Hash, 0, MaxStart*MaxStep)

	for step := 0; step < MaxStep; step++ {
		for startState := 0; startState < MaxStart; startState++ {
			if step == 0 && startState == 1 { // unlucky case where a bad proof would be accepted
				continue
			}
			hashGens = append(hashGens, test_vector_utils.NewMessageCounterGenerator(startState, step))
		}
	}

	for _, poly := range polys {
		for _, hashGen := range hashGens {
			assert.NoError(t, testSumcheckSingleClaimMultilin(poly, hashGen),
				"failed with poly %v and hashGen %v", poly, hashGen())
		}
	}
}
