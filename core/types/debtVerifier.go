/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package types

// DebtVerifier interface
type DebtVerifier interface {
	// ValidateDebt validate debt
	// returns packed whether debt is packed
	// returns confirmed whether debt is confirmed
	// returns retErr error info
	ValidateDebt(debt *Debt) (packed bool, confirmed bool, err error)

	// IfDebtPacked
	// returns packed whether debt is packed
	// returns confirmed whether debt is confirmed
	// returns retErr error info
	IfDebtPacked(debt *Debt) (packed bool, confirmed bool, err error)
}

type TestVerifier struct {
	packed    bool
	confirmed bool
	err       error
}

func NewTestVerifier(p bool, c bool, err error) *TestVerifier {
	return &TestVerifier{
		packed:    p,
		confirmed: c,
		err:       err,
	}
}

func (v *TestVerifier) ValidateDebt(debt *Debt) (packed bool, confirmed bool, err error) {
	return v.packed, v.confirmed, v.err
}

func (v *TestVerifier) IfDebtPacked(debt *Debt) (packed bool, confirmed bool, err error) {
	return v.packed, v.confirmed, v.err
}

type TestVerifierWithFunc struct {
	fun func(debt *Debt) (bool, bool, error)
}

func NewTestVerifierWithFunc(fun func(debt *Debt) (bool, bool, error)) *TestVerifierWithFunc {
	return &TestVerifierWithFunc{
		fun: fun,
	}
}

func (v *TestVerifierWithFunc) ValidateDebt(debt *Debt) (packed bool, confirmed bool, err error) {
	return v.fun(debt)
}

func (v *TestVerifierWithFunc) IfDebtPacked(debt *Debt) (packed bool, confirmed bool, err error) {
	return v.fun(debt)
}
