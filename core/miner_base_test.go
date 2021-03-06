package core

import "testing"

func Test_GetValidTicketsByHeight(t *testing.T) {
	tickets := getValidTicketsByHeight(1)
	if tickets != 8 {
		t.Fatalf("except 8 but got %d", tickets)
	}
	tickets = getValidTicketsByHeight(adjustWeightPeriod * 3)
	if tickets != 6 {
		t.Fatalf("except 6 but got %d", tickets)
	}
	tickets = getValidTicketsByHeight(adjustWeightPeriod * 3 * 2)
	if tickets != 4 {
		t.Fatalf("except 4 but got %d", tickets)
	}

	tickets = getValidTicketsByHeight(adjustWeightPeriod * 3 * 3)
	if tickets != 2 {
		t.Fatalf("except 2 but got %d", tickets)
	}
	tickets = getValidTicketsByHeight(adjustWeightPeriod * 3 * 4)
	if tickets != 1 {
		t.Fatalf("except 1 but got %d", tickets)
	}

	tickets = getValidTicketsByHeight(adjustWeightPeriod * 3 * 5)
	if tickets != 1 {
		t.Fatalf("except 1 but got %d", tickets)
	}

	tickets = getValidTicketsByHeight(adjustWeightPeriod * 3 * 6)
	if tickets != 1 {
		t.Fatalf("except 8 but got %d", tickets)
	}
}
