package bp

import (
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"math/big"
)

type MultiRangeProof struct {
	Comms []ECPoint
	A     ECPoint
	S     ECPoint
	T1    ECPoint
	T2    ECPoint
	Tau   *big.Int
	Th    *big.Int
	Mu    *big.Int
	IPP   InnerProdArg

	// challenges
	Cy *big.Int
	Cz *big.Int
	Cx *big.Int
}

// Calculates (aL - z*1^n) + sL*x
func CalculateLMRP(aL, sL []*big.Int, z, x *big.Int) []*big.Int {
	result := make([]*big.Int, len(aL))

	tmp1 := VectorAddScalar(aL, new(big.Int).Neg(z))
	tmp2 := ScalarVectorMul(sL, x)

	result = VectorAdd(tmp1, tmp2)

	return result
}

func CalculateRMRP(aR, sR, y, zTimesTwo []*big.Int, z, x *big.Int) []*big.Int {
	if len(aR) != len(sR) || len(aR) != len(y) || len(y) != len(zTimesTwo) {
		fmt.Println("CalculateR: Uh oh! Arrays not of the same length")
		fmt.Printf("len(aR): %d\n", len(aR))
		fmt.Printf("len(sR): %d\n", len(sR))
		fmt.Printf("len(y): %d\n", len(y))
		fmt.Printf("len(po2): %d\n", len(zTimesTwo))
	}

	result := make([]*big.Int, len(aR))

	tmp11 := VectorAddScalar(aR, z)
	tmp12 := ScalarVectorMul(sR, x)
	tmp1 := VectorHadamard(y, VectorAdd(tmp11, tmp12))

	result = VectorAdd(tmp1, zTimesTwo)

	return result
}

/*
DeltaMRP is a helper function that is used in the multi range proof
\delta(y, z) = (z-z^2)<1^n, y^n> - \sum_j z^3+j<1^n, 2^n>
*/

func DeltaMRP(y []*big.Int, z *big.Int, m int) *big.Int {
	result := big.NewInt(0)

	// (z-z^2)<1^n, y^n>
	z2 := new(big.Int).Mod(new(big.Int).Mul(z, z), EC.N)
	t1 := new(big.Int).Mod(new(big.Int).Sub(z, z2), EC.N)
	t2 := new(big.Int).Mod(new(big.Int).Mul(t1, VectorSum(y)), EC.N)

	// \sum_j z^3+j<1^n, 2^n>
	// <1^n, 2^n> = 2^n - 1
	po2sum := new(big.Int).Sub(new(big.Int).Exp(big.NewInt(2), big.NewInt(int64(EC.V/m)), EC.N), big.NewInt(1))
	t3 := big.NewInt(0)

	for j := 0; j < m; j++ {
		zp := new(big.Int).Exp(z, big.NewInt(3+int64(j)), EC.N)
		tmp1 := new(big.Int).Mod(new(big.Int).Mul(zp, po2sum), EC.N)
		t3 = new(big.Int).Mod(new(big.Int).Add(t3, tmp1), EC.N)
	}

	result = new(big.Int).Mod(new(big.Int).Sub(t2, t3), EC.N)

	return result
}

/*
MultiRangeProof Prove
Takes in a list of values and provides an aggregate
range proof for all the values.
changes:
 all values are concatenated
 r(x) is computed differently
 tau_x calculation is different
 delta calculation is different
{(g, h \in G, \textbf{V} \in G^m ; \textbf{v, \gamma} \in Z_p^m) :
	V_j = h^{\gamma_j}g^{v_j} \wedge v_j \in [0, 2^n - 1] \forall j \in [1, m]}
*/
func MRPProve(values []*big.Int) MultiRangeProof {
	// EC.V has the total number of values and bits we can support

	MRPResult := MultiRangeProof{}

	m := len(values)
	bitsPerValue := EC.V / m

	// we concatenate the binary representation of the values

	PowerOfTwos := PowerVector(bitsPerValue, big.NewInt(2))

	Comms := make([]ECPoint, m)
	gammas := make([]*big.Int, m)
	aLConcat := make([]*big.Int, EC.V)
	aRConcat := make([]*big.Int, EC.V)

	for j := range values {
		v := values[j]
		if v.Cmp(big.NewInt(0)) == -1 {
			panic("Value is below range! Not proving")
		}

		if v.Cmp(new(big.Int).Exp(big.NewInt(2), big.NewInt(int64(bitsPerValue)), EC.N)) == 1 {
			panic("Value is above range! Not proving.")
		}

		gamma, err := rand.Int(rand.Reader, EC.N)
		check(err)
		Comms[j] = EC.G.Mult(v).Add(EC.H.Mult(gamma))
		gammas[j] = gamma

		// break up v into its bitwise representation
		aL := reverse(StrToBigIntArray(PadLeft(fmt.Sprintf("%b", v), "0", bitsPerValue)))
		aR := VectorAddScalar(aL, big.NewInt(-1))

		for i := range aR {
			aLConcat[bitsPerValue*j+i] = aL[i]
			aRConcat[bitsPerValue*j+i] = aR[i]
		}
	}

	MRPResult.Comms = Comms

	alpha, err := rand.Int(rand.Reader, EC.N)
	check(err)

	A := TwoVectorPCommitWithGens(EC.BPG, EC.BPH, aLConcat, aRConcat).Add(EC.H.Mult(alpha))
	MRPResult.A = A

	sL := RandVector(EC.V)
	sR := RandVector(EC.V)

	rho, err := rand.Int(rand.Reader, EC.N)
	check(err)

	S := TwoVectorPCommitWithGens(EC.BPG, EC.BPH, sL, sR).Add(EC.H.Mult(rho))
	MRPResult.S = S

	chal1s256 := sha256.Sum256([]byte(A.X.String() + A.Y.String()))
	cy := new(big.Int).SetBytes(chal1s256[:])
	MRPResult.Cy = cy

	chal2s256 := sha256.Sum256([]byte(S.X.String() + S.Y.String()))
	cz := new(big.Int).SetBytes(chal2s256[:])
	MRPResult.Cz = cz

	zPowersTimesTwoVec := make([]*big.Int, EC.V)
	for j := 0; j < m; j++ {
		zp := new(big.Int).Exp(cz, big.NewInt(2+int64(j)), EC.N)
		for i := 0; i < bitsPerValue; i++ {
			zPowersTimesTwoVec[j*bitsPerValue+i] = new(big.Int).Mod(new(big.Int).Mul(PowerOfTwos[i], zp), EC.N)
		}
	}

	//fmt.Println(zPowersTimesTwoVec)

	// need to generate l(X), r(X), and t(X)=<l(X),r(X)>

	/*
			Java code on how to calculate t1 and t2
		        //z^Q
		        FieldVector zs = FieldVector.from(VectorX.iterate(m, z.pow(2), z::multiply).map(bi -> bi.mod(q)), q);
		        //2^n
		        VectorX<BigInteger> twoVector = VectorX.iterate(bitsPerNumber, BigInteger.ONE, bi -> bi.shiftLeft(1));
		        FieldVector twos = FieldVector.from(twoVector, q);
		        //2^n \cdot z || 2^n \cdot z^2 ...
		        FieldVector twoTimesZs = FieldVector.from(zs.getVector().flatMap(twos::times), q);
		        //l(X)
		        FieldVector l0 = aL.add(z.negate());
		        FieldVector l1 = sL;
		        FieldVectorPolynomial lPoly = new FieldVectorPolynomial(l0, l1);
		        //r(X)
		        FieldVector r0 = ys.hadamard(aR.add(z)).add(twoTimesZs);
		        FieldVector r1 = sR.hadamard(ys);
				FieldVectorPolynomial rPoly = new FieldVectorPolynomial(r0, r1);
	*/
	PowerOfCY := PowerVector(EC.V, cy)
	// fmt.Println(PowerOfCY)
	l0 := VectorAddScalar(aLConcat, new(big.Int).Neg(cz))
	l1 := sL
	r0 := VectorAdd(
		VectorHadamard(
			PowerOfCY,
			VectorAddScalar(aRConcat, cz)),
		zPowersTimesTwoVec)
	r1 := VectorHadamard(sR, PowerOfCY)

	//calculate t0
	vz2 := big.NewInt(0)
	z2 := new(big.Int).Mod(new(big.Int).Mul(cz, cz), EC.N)
	PowerOfCZ := PowerVector(m, cz)
	for j := 0; j < m; j++ {
		vz2 = new(big.Int).Add(vz2,
			new(big.Int).Mul(
				PowerOfCZ[j],
				new(big.Int).Mul(values[j], z2)))
		vz2 = new(big.Int).Mod(vz2, EC.N)
	}

	t0 := new(big.Int).Mod(new(big.Int).Add(vz2, DeltaMRP(PowerOfCY, cz, m)), EC.N)

	t1 := new(big.Int).Mod(new(big.Int).Add(InnerProduct(l1, r0), InnerProduct(l0, r1)), EC.N)
	t2 := InnerProduct(l1, r1)

	// given the t_i values, we can generate commitments to them
	tau1, err := rand.Int(rand.Reader, EC.N)
	check(err)
	tau2, err := rand.Int(rand.Reader, EC.N)
	check(err)

	T1 := EC.G.Mult(t1).Add(EC.H.Mult(tau1)) //commitment to t1
	T2 := EC.G.Mult(t2).Add(EC.H.Mult(tau2)) //commitment to t2

	MRPResult.T1 = T1
	MRPResult.T2 = T2

	chal3s256 := sha256.Sum256([]byte(T1.X.String() + T1.Y.String() + T2.X.String() + T2.Y.String()))
	cx := new(big.Int).SetBytes(chal3s256[:])

	MRPResult.Cx = cx

	left := CalculateLMRP(aLConcat, sL, cz, cx)
	right := CalculateRMRP(aRConcat, sR, PowerOfCY, zPowersTimesTwoVec, cz, cx)

	thatPrime := new(big.Int).Mod( // t0 + t1*x + t2*x^2
		new(big.Int).Add(t0, new(big.Int).Add(new(big.Int).Mul(t1, cx), new(big.Int).Mul(new(big.Int).Mul(cx, cx), t2))), EC.N)

	that := InnerProduct(left, right) // NOTE: BP Java implementation calculates this from the t_i

	// thatPrime and that should be equal
	if thatPrime.Cmp(that) != 0 {
		fmt.Println("Proving -- Uh oh! Two diff ways to compute same value not working")
		fmt.Printf("\tthatPrime = %s\n", thatPrime.String())
		fmt.Printf("\tthat = %s \n", that.String())
	}

	MRPResult.Th = that

	vecRandomnessTotal := big.NewInt(0)
	for j := 0; j < m; j++ {
		zp := new(big.Int).Exp(cz, big.NewInt(2+int64(j)), EC.N)
		tmp1 := new(big.Int).Mul(gammas[j], zp)
		vecRandomnessTotal = new(big.Int).Mod(new(big.Int).Add(vecRandomnessTotal, tmp1), EC.N)
	}
	//fmt.Println(vecRandomnessTotal)
	taux1 := new(big.Int).Mod(new(big.Int).Mul(tau2, new(big.Int).Mul(cx, cx)), EC.N)
	taux2 := new(big.Int).Mod(new(big.Int).Mul(tau1, cx), EC.N)
	taux := new(big.Int).Mod(new(big.Int).Add(taux1, new(big.Int).Add(taux2, vecRandomnessTotal)), EC.N)

	MRPResult.Tau = taux

	mu := new(big.Int).Mod(new(big.Int).Add(alpha, new(big.Int).Mul(rho, cx)), EC.N)
	MRPResult.Mu = mu

	HPrime := make([]ECPoint, len(EC.BPH))

	for i := range HPrime {
		HPrime[i] = EC.BPH[i].Mult(new(big.Int).ModInverse(PowerOfCY[i], EC.N))
	}

	P := TwoVectorPCommitWithGens(EC.BPG, HPrime, left, right)
	//fmt.Println(P)

	MRPResult.IPP = InnerProductProve(left, right, that, P, EC.U, EC.BPG, HPrime)

	return MRPResult
}

/*
MultiRangeProof Verify
Takes in a MultiRangeProof and verifies its correctness
*/
func MRPVerify(mrp MultiRangeProof) bool {
	m := len(mrp.Comms)
	bitsPerValue := EC.V / m

	//changes:
	// check 1 changes since it includes all commitments
	// check 2 commitment generation is also different

	// verify the challenges
	chal1s256 := sha256.Sum256([]byte(mrp.A.X.String() + mrp.A.Y.String()))
	cy := new(big.Int).SetBytes(chal1s256[:])
	if cy.Cmp(mrp.Cy) != 0 {
		fmt.Println("MRPVerify - Challenge Cy failing!")
		return false
	}
	chal2s256 := sha256.Sum256([]byte(mrp.S.X.String() + mrp.S.Y.String()))
	cz := new(big.Int).SetBytes(chal2s256[:])
	if cz.Cmp(mrp.Cz) != 0 {
		fmt.Println("MRPVerify - Challenge Cz failing!")
		return false
	}
	chal3s256 := sha256.Sum256([]byte(mrp.T1.X.String() + mrp.T1.Y.String() + mrp.T2.X.String() + mrp.T2.Y.String()))
	cx := new(big.Int).SetBytes(chal3s256[:])
	if cx.Cmp(mrp.Cx) != 0 {
		fmt.Println("RPVerify - Challenge Cx failing!")
		return false
	}

	// given challenges are correct, very range proof
	PowersOfY := PowerVector(EC.V, cy)

	// t_hat * G + tau * H
	lhs := EC.G.Mult(mrp.Th).Add(EC.H.Mult(mrp.Tau))

	// z^2 * \bold{z}^m \bold{V} + delta(y,z) * G + x * T1 + x^2 * T2
	CommPowers := EC.Zero()
	PowersOfZ := PowerVector(m, cz)
	z2 := new(big.Int).Mod(new(big.Int).Mul(cz, cz), EC.N)

	for j := 0; j < m; j++ {
		CommPowers = CommPowers.Add(mrp.Comms[j].Mult(new(big.Int).Mul(z2, PowersOfZ[j])))
	}

	rhs := EC.G.Mult(DeltaMRP(PowersOfY, cz, m)).Add(
		mrp.T1.Mult(cx)).Add(
		mrp.T2.Mult(new(big.Int).Mul(cx, cx))).Add(CommPowers)

	if !lhs.Equal(rhs) {
		fmt.Println("MRPVerify - Uh oh! Check line (63) of verification")
		fmt.Println(rhs)
		fmt.Println(lhs)
		return false
	}

	tmp1 := EC.Zero()
	zneg := new(big.Int).Mod(new(big.Int).Neg(cz), EC.N)
	for i := range EC.BPG {
		tmp1 = tmp1.Add(EC.BPG[i].Mult(zneg))
	}

	PowerOfTwos := PowerVector(bitsPerValue, big.NewInt(2))
	tmp2 := EC.Zero()
	// generate h'
	HPrime := make([]ECPoint, len(EC.BPH))

	for i := range HPrime {
		mi := new(big.Int).ModInverse(PowersOfY[i], EC.N)
		HPrime[i] = EC.BPH[i].Mult(mi)
	}

	for j := 0; j < m; j++ {
		for i := 0; i < bitsPerValue; i++ {
			val1 := new(big.Int).Mul(cz, PowersOfY[j*bitsPerValue+i])
			zp := new(big.Int).Exp(cz, big.NewInt(2+int64(j)), EC.N)
			val2 := new(big.Int).Mod(new(big.Int).Mul(zp, PowerOfTwos[i]), EC.N)
			tmp2 = tmp2.Add(HPrime[j*bitsPerValue+i].Mult(new(big.Int).Add(val1, val2)))
		}
	}

	// without subtracting this value should equal muCH + l[i]G[i] + r[i]H'[i]
	// we want to make sure that the innerproduct checks out, so we subtract it
	P := mrp.A.Add(mrp.S.Mult(cx)).Add(tmp1).Add(tmp2).Add(EC.H.Mult(mrp.Mu).Neg())
	//fmt.Println(P)

	if !InnerProductVerifyFast(mrp.Th, P, EC.U, EC.BPG, HPrime, mrp.IPP) {
		fmt.Println("MRPVerify - Uh oh! Check line (65) of verification!")
		return false
	}

	return true
}
