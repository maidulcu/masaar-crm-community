package repo

import (
	"testing"

	"github.com/google/uuid"
)

func TestSortLeaderboardByMetric(t *testing.T) {
	id1 := uuid.New()
	id2 := uuid.New()
	id3 := uuid.New()

	agents := []AgentKPIs{
		{AgentID: id1, AgentName: "Agent A", DealsWon: 2, Revenue: 1000, ListingsAdded: 5, LeadsConverted: 1, LeadsAssigned: 10},
		{AgentID: id2, AgentName: "Agent B", DealsWon: 5, Revenue: 500, ListingsAdded: 2, LeadsConverted: 4, LeadsAssigned: 5},
		{AgentID: id3, AgentName: "Agent C", DealsWon: 3, Revenue: 2000, ListingsAdded: 10, LeadsConverted: 2, LeadsAssigned: 8},
	}

	t.Run("sort by revenue (default)", func(t *testing.T) {
		board := append([]AgentKPIs(nil), agents...)
		SortLeaderboardByMetric(board, "revenue")
		if board[0].AgentID != id3 || board[1].AgentID != id1 || board[2].AgentID != id2 {
			t.Errorf("unexpected order for revenue: %+v", board)
		}
	})

	t.Run("sort by deals_won", func(t *testing.T) {
		board := append([]AgentKPIs(nil), agents...)
		SortLeaderboardByMetric(board, "deals_won")
		if board[0].AgentID != id2 || board[1].AgentID != id3 || board[2].AgentID != id1 {
			t.Errorf("unexpected order for deals_won: %+v", board)
		}
	})

	t.Run("sort by listings_added", func(t *testing.T) {
		board := append([]AgentKPIs(nil), agents...)
		SortLeaderboardByMetric(board, "listings_added")
		if board[0].AgentID != id3 || board[1].AgentID != id1 || board[2].AgentID != id2 {
			t.Errorf("unexpected order for listings_added: %+v", board)
		}
	})

	t.Run("sort by leads_converted", func(t *testing.T) {
		board := append([]AgentKPIs(nil), agents...)
		SortLeaderboardByMetric(board, "leads_converted")
		if board[0].AgentID != id2 || board[1].AgentID != id3 || board[2].AgentID != id1 {
			t.Errorf("unexpected order for leads_converted: %+v", board)
		}
	})

	t.Run("sort by leads_assigned", func(t *testing.T) {
		board := append([]AgentKPIs(nil), agents...)
		SortLeaderboardByMetric(board, "leads_assigned")
		if board[0].AgentID != id1 || board[1].AgentID != id3 || board[2].AgentID != id2 {
			t.Errorf("unexpected order for leads_assigned: %+v", board)
		}
	})

	t.Run("empty board", func(t *testing.T) {
		var board []AgentKPIs
		SortLeaderboardByMetric(board, "revenue")
		if len(board) != 0 {
			t.Errorf("expected empty board, got %d items", len(board))
		}
	})
}
