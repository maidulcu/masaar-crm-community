package jobs

import "testing"

func TestAlertPoint(t *testing.T) {
	for daysLeft, want := range map[int]int{45: -1, 31: -1, 30: 30, 12: 30, 7: 7, 1: 7, 0: 7, -1: 0, -40: 0} {
		if got := alertPoint(daysLeft); got != want {
			t.Errorf("alertPoint(%d) = %d, want %d", daysLeft, got, want)
		}
	}
}
