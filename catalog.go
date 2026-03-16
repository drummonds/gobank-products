package gbp

// EasyAccess returns an instant-access savings product.
func EasyAccess() *Product {
	return &Product{
		ID:     "easy-access",
		Name:   "Easy Access",
		Family: FamilySavings,
		Features: []Feature{
			StatusLifecycle{},
			DepositAcceptance{},
			WithdrawalProcessing{},
			InterestAccrual{},
		},
		Defaults: map[string]string{
			"annual_rate": "0.015",
		},
	}
}

// FixedTerm returns a fixed-term savings product.
func FixedTerm() *Product {
	return &Product{
		ID:     "fixed-term",
		Name:   "Fixed Term",
		Family: FamilySavings,
		Features: []Feature{
			StatusLifecycle{},
			DepositAcceptance{},
			TermLock{},
			WithdrawalProcessing{},
			InterestAccrual{},
		},
		Defaults: map[string]string{
			"annual_rate": "0.040",
		},
	}
}

// ISA returns a tax-free ISA savings product.
func ISA() *Product {
	return &Product{
		ID:     "isa",
		Name:   "ISA",
		Family: FamilySavings,
		Features: []Feature{
			StatusLifecycle{},
			ISAWrapper{},
			DepositAcceptance{},
			WithdrawalProcessing{},
			InterestAccrual{},
		},
		Defaults: map[string]string{
			"annual_rate":   "0.035",
			"isa_allowance": "2000000", // £20,000.00 in pence
		},
	}
}

// PersonalLoan returns a personal loan product.
func PersonalLoan() *Product {
	return &Product{
		ID:     "personal-loan",
		Name:   "Personal Loan",
		Family: FamilyLending,
		Features: []Feature{
			StatusLifecycle{},
			DepositAcceptance{},
			InterestAccrual{},
			RepaymentSchedule{},
		},
		Defaults: map[string]string{
			"annual_rate": "0.069",
		},
	}
}

// Mortgage returns a residential mortgage product.
func Mortgage() *Product {
	return &Product{
		ID:     "mortgage",
		Name:   "Mortgage",
		Family: FamilyLending,
		Features: []Feature{
			StatusLifecycle{},
			DepositAcceptance{},
			InterestAccrual{},
			RepaymentSchedule{},
		},
		Defaults: map[string]string{
			"annual_rate": "0.045",
		},
	}
}

// Overdraft returns an overdraft facility product.
func Overdraft() *Product {
	return &Product{
		ID:     "overdraft",
		Name:   "Overdraft",
		Family: FamilyLending,
		Features: []Feature{
			StatusLifecycle{},
			DepositAcceptance{},
			OverdraftFacility{},
			InterestAccrual{},
		},
		Defaults: map[string]string{
			"annual_rate":     "0.159",
			"overdraft_limit": "100000", // £1,000.00 in pence
		},
	}
}
