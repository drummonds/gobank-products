package gbp_test

import (
	"testing"

	gbp "codeberg.org/hum3/gobank-products"
	"codeberg.org/hum3/gobank-products/testkit"
)

func TestStatusLifecycle_OpenActivates(t *testing.T) {
	testkit.NewScenario(t).
		WithProduct(gbp.EasyAccess()).
		OpenAccount("easy-access", "Liability:Savings:alice").
		AssertStatus("Liability:Savings:alice", gbp.StatusActive)
}

func TestStatusLifecycle_Close(t *testing.T) {
	testkit.NewScenario(t).
		WithProduct(gbp.EasyAccess()).
		OpenAccount("easy-access", "Liability:Savings:alice").
		CloseAccount("Liability:Savings:alice").
		AssertStatus("Liability:Savings:alice", gbp.StatusClosed)
}
