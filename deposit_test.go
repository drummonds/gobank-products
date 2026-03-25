package gbp_test

import (
	"testing"

	gbp "codeberg.org/hum3/gobank-products"
	"codeberg.org/hum3/gobank-products/testkit"
)

func TestDeposit_Basic(t *testing.T) {
	testkit.NewScenario(t).
		WithProduct(gbp.EasyAccess()).
		OpenAccount("easy-access", "Liability:Savings:alice").
		Deposit("Liability:Savings:alice", 100000).
		AssertBalance("Liability:Savings:alice", -100000) // credit-normal
}

func TestDeposit_Multiple(t *testing.T) {
	testkit.NewScenario(t).
		WithProduct(gbp.EasyAccess()).
		OpenAccount("easy-access", "Liability:Savings:alice").
		Deposit("Liability:Savings:alice", 50000).
		Deposit("Liability:Savings:alice", 30000).
		AssertBalance("Liability:Savings:alice", -80000) // credit-normal
}
