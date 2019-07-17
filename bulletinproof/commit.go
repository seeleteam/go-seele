package bp

import (
	"crypto/rand"
	"math/big"
)

/**
Vector Pedersen Commitment
Given an array of values, we commit the array with different generators
for each element and for each randomness.
*/

func PedersonCommit(value []*big.Int) (ECPoint, []*big.Int) {
	R := make([]*big.Int, EC.V)
	commitment := EC.Zero()
	for i := 0; i < EC.V; i++ {
		r, err := rand.Int(rand.Reader, EC.N)
		check(err)
		R[i] = r
		modValue := new(big.Int).Mod(value[i], EC.N)
		//mG, rH
		lhsX, lhsY := EC.C.ScalarMult(EC.BPG[i].X, EC.BPG[i].Y, modValue.Bytes())
		rhsX, rhsY := EC.C.ScalarMult(EC.BPH[i].X, EC.BPH[i].Y, r.Bytes())
		commitment = commitment.Add(ECPoint{lhsX, lhsY}).Add(ECPoint{rhsX, rhsY})
	}
	return commitment, R
}
