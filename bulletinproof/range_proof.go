package bp

import (
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"math/big"
)

type RangeProof struct {
	Comm ECPoint
	A    ECPoint
	S    ECPoint
	T1   ECPoint
	T2   ECPoint
	Tau  *big.Int
	Th   *big.Int
	Mu   *big.Int
	IPP  InnerProdArg

	// challenges
	Cy *big.Int
	Cz *big.Int
	Cx *big.Int
}

/*
Delta is a helper function that is used in the range proof
\delta(y, z) = (z-z^2)<1^n, y^n> - z^3<1^n, 2^n>
*/

func Delta(y []*big.Int, z *big.Int) *big.Int {
	result := big.NewInt(0)

	// (z-z^2)<1^n, y^n>
	z2 := new(big.Int).Mod(new(big.Int).Mul(z, z), EC.N)
	t1 := new(big.Int).Mod(new(big.Int).Sub(z, z2), EC.N)
	t2 := new(big.Int).Mod(new(big.Int).Mul(t1, VectorSum(y)), EC.N)

	// z^3<1^n, 2^n>
	z3 := new(big.Int).Mod(new(big.Int).Mul(z2, z), EC.N)
	po2sum := new(big.Int).Sub(new(big.Int).Exp(big.NewInt(2), big.NewInt(int64(EC.V)), EC.N), big.NewInt(1))
	t3 := new(big.Int).Mod(new(big.Int).Mul(z3, po2sum), EC.N)

	result = new(big.Int).Mod(new(big.Int).Sub(t2, t3), EC.N)

	return result
}

// Calculates (aL - z*1^n) + sL*x
func CalculateL(aL, sL []*big.Int, z, x *big.Int) []*big.Int {
	result := make([]*big.Int, len(aL))

	tmp1 := VectorAddScalar(aL, new(big.Int).Neg(z))
	tmp2 := ScalarVectorMul(sL, x)

	result = VectorAdd(tmp1, tmp2)

	return result
}

func CalculateR(aR, sR, y, po2 []*big.Int, z, x *big.Int) []*big.Int {
	if len(aR) != len(sR) || len(aR) != len(y) || len(y) != len(po2) {
		fmt.Println("CalculateR: Uh oh! Arrays not of the same length")
		fmt.Printf("len(aR): %d\n", len(aR))
		fmt.Printf("len(sR): %d\n", len(sR))
		fmt.Printf("len(y): %d\n", len(y))
		fmt.Printf("len(po2): %d\n", len(po2))
	}

	result := make([]*big.Int, len(aR))

	z2 := new(big.Int).Exp(z, big.NewInt(2), EC.N)
	tmp11 := VectorAddScalar(aR, z)
	tmp12 := ScalarVectorMul(sR, x)
	tmp1 := VectorHadamard(y, VectorAdd(tmp11, tmp12))
	tmp2 := ScalarVectorMul(po2, z2)

	result = VectorAdd(tmp1, tmp2)

	return result
}

/*
RPProver : Range Proof Prove
Given a value v, provides a range proof that v is inside 0 to 2^64-1
*/
func RPProve(v *big.Int) RangeProof {

	rpresult := RangeProof{}

	PowerOfTwos := PowerVector(EC.V, big.NewInt(2))

	if v.Cmp(big.NewInt(0)) == -1 {
		panic("Value is below range! Not proving")
	}

	if v.Cmp(new(big.Int).Exp(big.NewInt(2), big.NewInt(int64(EC.V)), EC.N)) == 1 {
		panic("Value is above range! Not proving.")
	}

	gamma, err := rand.Int(rand.Reader, EC.N)
	check(err)
	comm := EC.G.Mult(v).Add(EC.H.Mult(gamma))
	rpresult.Comm = comm

	// break up v into its bitwise representation
	//aL := 0
	aL := reverse(StrToBigIntArray(PadLeft(fmt.Sprintf("%b", v), "0", EC.V)))
	aR := VectorAddScalar(aL, big.NewInt(-1))

	alpha, err := rand.Int(rand.Reader, EC.N)
	check(err)

	A := TwoVectorPCommitWithGens(EC.BPG, EC.BPH, aL, aR).Add(EC.H.Mult(alpha))
	rpresult.A = A

	sL := RandVector(EC.V)
	sR := RandVector(EC.V)

	rho, err := rand.Int(rand.Reader, EC.N)
	check(err)

	S := TwoVectorPCommitWithGens(EC.BPG, EC.BPH, sL, sR).Add(EC.H.Mult(rho))
	rpresult.S = S

	chal1s256 := sha256.Sum256([]byte(A.X.String() + A.Y.String()))
	cy := new(big.Int).SetBytes(chal1s256[:])

	rpresult.Cy = cy

	chal2s256 := sha256.Sum256([]byte(S.X.String() + S.Y.String()))
	cz := new(big.Int).SetBytes(chal2s256[:])

	rpresult.Cz = cz
	z2 := new(big.Int).Exp(cz, big.NewInt(2), EC.N)
	// need to generate l(X), r(X), and t(X)=<l(X),r(X)>

	/*
			Java code on how to calculate t1 and t2
				FieldVector ys = FieldVector.from(VectorX.iterate(n, BigInteger.ONE, y::multiply),q); //powers of y
			    FieldVector l0 = aL.add(z.negate());
		        FieldVector l1 = sL;
		        FieldVector twoTimesZSquared = twos.times(zSquared);
		        FieldVector r0 = ys.hadamard(aR.add(z)).add(twoTimesZSquared);
		        FieldVector r1 = sR.hadamard(ys);
		        BigInteger k = ys.sum().multiply(z.subtract(zSquared)).subtract(zCubed.shiftLeft(n).subtract(zCubed));
		        BigInteger t0 = k.add(zSquared.multiply(number));
		        BigInteger t1 = l1.innerPoduct(r0).add(l0.innerPoduct(r1));
		        BigInteger t2 = l1.innerPoduct(r1);
		   		PolyCommitment<T> polyCommitment = PolyCommitment.from(base, t0, VectorX.of(t1, t2));
	*/
	PowerOfCY := PowerVector(EC.V, cy)
	// fmt.Println(PowerOfCY)
	l0 := VectorAddScalar(aL, new(big.Int).Neg(cz))
	// l1 := sL
	r0 := VectorAdd(
		VectorHadamard(
			PowerOfCY,
			VectorAddScalar(aR, cz)),
		ScalarVectorMul(
			PowerOfTwos,
			z2))
	r1 := VectorHadamard(sR, PowerOfCY)

	//calculate t0
	t0 := new(big.Int).Mod(new(big.Int).Add(new(big.Int).Mul(v, z2), Delta(PowerOfCY, cz)), EC.N)

	t1 := new(big.Int).Mod(new(big.Int).Add(InnerProduct(sL, r0), InnerProduct(l0, r1)), EC.N)
	t2 := InnerProduct(sL, r1)

	// given the t_i values, we can generate commitments to them
	tau1, err := rand.Int(rand.Reader, EC.N)
	check(err)
	tau2, err := rand.Int(rand.Reader, EC.N)
	check(err)

	T1 := EC.G.Mult(t1).Add(EC.H.Mult(tau1)) //commitment to t1
	T2 := EC.G.Mult(t2).Add(EC.H.Mult(tau2)) //commitment to t2

	rpresult.T1 = T1
	rpresult.T2 = T2

	chal3s256 := sha256.Sum256([]byte(T1.X.String() + T1.Y.String() + T2.X.String() + T2.Y.String()))
	cx := new(big.Int).SetBytes(chal3s256[:])

	rpresult.Cx = cx

	left := CalculateL(aL, sL, cz, cx)
	right := CalculateR(aR, sR, PowerOfCY, PowerOfTwos, cz, cx)

	thatPrime := new(big.Int).Mod( // t0 + t1*x + t2*x^2
		new(big.Int).Add(
			t0,
			new(big.Int).Add(
				new(big.Int).Mul(
					t1, cx),
				new(big.Int).Mul(
					new(big.Int).Mul(cx, cx),
					t2))), EC.N)

	that := InnerProduct(left, right) // NOTE: BP Java implementation calculates this from the t_i

	// thatPrime and that should be equal
	if thatPrime.Cmp(that) != 0 {
		fmt.Println("Proving -- Uh oh! Two diff ways to compute same value not working")
		fmt.Printf("\tthatPrime = %s\n", thatPrime.String())
		fmt.Printf("\tthat = %s \n", that.String())
	}

	rpresult.Th = thatPrime

	taux1 := new(big.Int).Mod(new(big.Int).Mul(tau2, new(big.Int).Mul(cx, cx)), EC.N)
	taux2 := new(big.Int).Mod(new(big.Int).Mul(tau1, cx), EC.N)
	taux3 := new(big.Int).Mod(new(big.Int).Mul(z2, gamma), EC.N)
	taux := new(big.Int).Mod(new(big.Int).Add(taux1, new(big.Int).Add(taux2, taux3)), EC.N)

	rpresult.Tau = taux

	mu := new(big.Int).Mod(new(big.Int).Add(alpha, new(big.Int).Mul(rho, cx)), EC.N)
	rpresult.Mu = mu

	HPrime := make([]ECPoint, len(EC.BPH))

	for i := range HPrime {
		HPrime[i] = EC.BPH[i].Mult(new(big.Int).ModInverse(PowerOfCY[i], EC.N))
	}

	// for testing
	tmp1 := EC.Zero()
	zneg := new(big.Int).Mod(new(big.Int).Neg(cz), EC.N)
	for i := range EC.BPG {
		tmp1 = tmp1.Add(EC.BPG[i].Mult(zneg))
	}

	tmp2 := EC.Zero()
	for i := range HPrime {
		val1 := new(big.Int).Mul(cz, PowerOfCY[i])
		val2 := new(big.Int).Mul(new(big.Int).Mul(cz, cz), PowerOfTwos[i])
		tmp2 = tmp2.Add(HPrime[i].Mult(new(big.Int).Add(val1, val2)))
	}

	//P1 := A.Add(S.Mult(cx)).Add(tmp1).Add(tmp2).Add(EC.U.Mult(that)).Add(EC.H.Mult(mu).Neg())

	P := TwoVectorPCommitWithGens(EC.BPG, HPrime, left, right)
	//fmt.Println(P1)
	//fmt.Println(P2)

	rpresult.IPP = InnerProductProve(left, right, that, P, EC.U, EC.BPG, HPrime)

	return rpresult
}

func RPVerify(rp RangeProof) bool {
	// verify the challenges
	chal1s256 := sha256.Sum256([]byte(rp.A.X.String() + rp.A.Y.String()))
	cy := new(big.Int).SetBytes(chal1s256[:])
	if cy.Cmp(rp.Cy) != 0 {
		fmt.Println("RPVerify - Challenge Cy failing!")
		return false
	}
	chal2s256 := sha256.Sum256([]byte(rp.S.X.String() + rp.S.Y.String()))
	cz := new(big.Int).SetBytes(chal2s256[:])
	if cz.Cmp(rp.Cz) != 0 {
		fmt.Println("RPVerify - Challenge Cz failing!")
		return false
	}
	chal3s256 := sha256.Sum256([]byte(rp.T1.X.String() + rp.T1.Y.String() + rp.T2.X.String() + rp.T2.Y.String()))
	cx := new(big.Int).SetBytes(chal3s256[:])
	if cx.Cmp(rp.Cx) != 0 {
		fmt.Println("RPVerify - Challenge Cx failing!")
		return false
	}

	// given challenges are correct, very range proof
	PowersOfY := PowerVector(EC.V, cy)

	// t_hat * G + tau * H
	lhs := EC.G.Mult(rp.Th).Add(EC.H.Mult(rp.Tau))

	// z^2 * V + delta(y,z) * G + x * T1 + x^2 * T2
	rhs := rp.Comm.Mult(new(big.Int).Mul(cz, cz)).Add(
		EC.G.Mult(Delta(PowersOfY, cz))).Add(
		rp.T1.Mult(cx)).Add(
		rp.T2.Mult(new(big.Int).Mul(cx, cx)))

	if !lhs.Equal(rhs) {
		fmt.Println("RPVerify - Uh oh! Check line (63) of verification")
		fmt.Println(rhs)
		fmt.Println(lhs)
		return false
	}

	tmp1 := EC.Zero()
	zneg := new(big.Int).Mod(new(big.Int).Neg(cz), EC.N)
	for i := range EC.BPG {
		tmp1 = tmp1.Add(EC.BPG[i].Mult(zneg))
	}

	PowerOfTwos := PowerVector(EC.V, big.NewInt(2))
	tmp2 := EC.Zero()
	// generate h'
	HPrime := make([]ECPoint, len(EC.BPH))

	for i := range HPrime {
		mi := new(big.Int).ModInverse(PowersOfY[i], EC.N)
		HPrime[i] = EC.BPH[i].Mult(mi)
	}

	for i := range HPrime {
		val1 := new(big.Int).Mul(cz, PowersOfY[i])
		val2 := new(big.Int).Mul(new(big.Int).Mul(cz, cz), PowerOfTwos[i])
		tmp2 = tmp2.Add(HPrime[i].Mult(new(big.Int).Add(val1, val2)))
	}

	// without subtracting this value should equal muCH + l[i]G[i] + r[i]H'[i]
	// we want to make sure that the innerproduct checks out, so we subtract it
	P := rp.A.Add(rp.S.Mult(cx)).Add(tmp1).Add(tmp2).Add(EC.H.Mult(rp.Mu).Neg())
	//fmt.Println(P)

	if !InnerProductVerifyFast(rp.Th, P, EC.U, EC.BPG, HPrime, rp.IPP) {
		fmt.Println("RPVerify - Uh oh! Check line (65) of verification!")
		return false
	}

	return true
}
