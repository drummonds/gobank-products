package gbp_test

import (
	"testing"

	gbp "git.bytestone.uk/hum3/gobank-products"
	"git.bytestone.uk/hum3/gobank-products/testkit"
)

func TestWithdrawal_Basic(t *testing.T) {
	testkit.NewScenario(t).
		WithProduct(gbp.EasyAccess()).
		OpenAccount("easy-access", "Liability:Savings:alice").
		Deposit("Liability:Savings:alice", 100000).
		Withdraw("Liability:Savings:alice", 30000).
		AssertBalance("Liability:Savings:alice", -70000) // credit-normal
}

func TestWithdrawal_InsufficientFunds(t *testing.T) {
	testkit.NewScenario(t).
		WithProduct(gbp.EasyAccess()).
		OpenAccount("easy-access", "Liability:Savings:alice").
		Deposit("Liability:Savings:alice", 10000).
		WithdrawExpectError("Liability:Savings:alice", 50000)
}
