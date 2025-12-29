package sqlite_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"github.com/vytor/chessflash/internal/models"
	"github.com/vytor/chessflash/internal/repository"
	"github.com/vytor/chessflash/internal/repository/sqlite"
	"github.com/vytor/chessflash/internal/testutil"
)

type GameRepositorySuite struct {
	suite.Suite
	db   *sql.DB
	repo repository.GameRepository
}

func (s *GameRepositorySuite) SetupTest() {
	s.db = testutil.NewTestDB(s.T())
	s.repo = sqlite.NewGameRepository(s.db)
}

func (s *GameRepositorySuite) TearDownTest() {
	testutil.MustClose(s.T(), s.db)
}

func (s *GameRepositorySuite) TestInsertAndGet() {
	ctx := context.Background()

	// First create a profile
	_, err := s.db.ExecContext(ctx, `INSERT INTO profiles (username) VALUES (?)`, "testuser")
	s.Require().NoError(err)

	var profileID int64
	err = s.db.QueryRowContext(ctx, `SELECT id FROM profiles WHERE username = ?`, "testuser").Scan(&profileID)
	s.Require().NoError(err)

	game := models.Game{
		ProfileID:      profileID,
		ChessComID:     "test123",
		PGN:            "[Event \"Test\"]\n1. e4 e5",
		TimeClass:      "blitz",
		Result:         "win",
		PlayedAs:       "white",
		Opponent:       "opponent1",
		PlayerRating:   1500,
		OpponentRating: 1600,
		PlayedAt:       time.Now(),
		ECOCode:        "B20",
		OpeningName:    "Sicilian Defense",
		AnalysisStatus: "pending",
	}

	id, err := s.repo.Insert(ctx, game)
	s.Require().NoError(err)
	s.Assert().Greater(id, int64(0))

	retrieved, err := s.repo.Get(ctx, id)
	s.Require().NoError(err)
	s.Assert().Equal("opponent1", retrieved.Opponent)
	s.Assert().Equal("blitz", retrieved.TimeClass)
	s.Assert().Equal("win", retrieved.Result)
	s.Assert().Equal("test123", retrieved.ChessComID)
}

func (s *GameRepositorySuite) TestGet_NotFound() {
	ctx := context.Background()
	game, err := s.repo.Get(ctx, 99999)
	s.Assert().Error(err)
	s.Assert().Nil(game)
}

func (s *GameRepositorySuite) TestInsertBatch() {
	ctx := context.Background()

	// Create profile
	_, err := s.db.ExecContext(ctx, `INSERT INTO profiles (username) VALUES (?)`, "testuser")
	s.Require().NoError(err)

	var profileID int64
	err = s.db.QueryRowContext(ctx, `SELECT id FROM profiles WHERE username = ?`, "testuser").Scan(&profileID)
	s.Require().NoError(err)

	games := []models.Game{
		{
			ProfileID:      profileID,
			ChessComID:     "game1",
			PGN:            "test pgn 1",
			TimeClass:      "blitz",
			Result:         "win",
			PlayedAs:       "white",
			Opponent:       "opp1",
			PlayedAt:       time.Now(),
			AnalysisStatus: "pending",
		},
		{
			ProfileID:      profileID,
			ChessComID:     "game2",
			PGN:            "test pgn 2",
			TimeClass:      "rapid",
			Result:         "loss",
			PlayedAs:       "black",
			Opponent:       "opp2",
			PlayedAt:       time.Now(),
			AnalysisStatus: "pending",
		},
	}

	ids, err := s.repo.InsertBatch(ctx, games)
	s.Require().NoError(err)
	s.Assert().Len(ids, 2)
	for _, id := range ids {
		s.Assert().Greater(id, int64(0))
	}
}

func (s *GameRepositorySuite) TestList_WithFilters() {
	ctx := context.Background()

	// Create profile
	_, err := s.db.ExecContext(ctx, `INSERT INTO profiles (username) VALUES (?)`, "testuser")
	s.Require().NoError(err)

	var profileID int64
	err = s.db.QueryRowContext(ctx, `SELECT id FROM profiles WHERE username = ?`, "testuser").Scan(&profileID)
	s.Require().NoError(err)

	// Insert test games
	games := []models.Game{
		{
			ProfileID:      profileID,
			ChessComID:     "game1",
			PGN:            "test",
			TimeClass:      "blitz",
			Result:         "win",
			PlayedAs:       "white",
			Opponent:       "opp1",
			PlayedAt:       time.Now(),
			AnalysisStatus: "pending",
		},
		{
			ProfileID:      profileID,
			ChessComID:     "game2",
			PGN:            "test",
			TimeClass:      "rapid",
			Result:         "loss",
			PlayedAs:       "black",
			Opponent:       "opp2",
			PlayedAt:       time.Now(),
			AnalysisStatus: "pending",
		},
	}

	_, err = s.repo.InsertBatch(ctx, games)
	s.Require().NoError(err)

	// Test filter by time class
	filter := models.GameFilter{
		ProfileID: profileID,
		TimeClass: "blitz",
		Limit:     10,
		Offset:    0,
	}

	result, err := s.repo.List(ctx, filter)
	s.Require().NoError(err)
	s.Assert().Len(result, 1)
	s.Assert().Equal("blitz", result[0].TimeClass)

	// Test filter by result
	filter = models.GameFilter{
		ProfileID: profileID,
		Result:    "win",
		Limit:     10,
		Offset:    0,
	}

	result, err = s.repo.List(ctx, filter)
	s.Require().NoError(err)
	s.Assert().Len(result, 1)
	s.Assert().Equal("win", result[0].Result)
}

func (s *GameRepositorySuite) TestUpdateStatus() {
	ctx := context.Background()

	// Create profile and game
	_, err := s.db.ExecContext(ctx, `INSERT INTO profiles (username) VALUES (?)`, "testuser")
	s.Require().NoError(err)

	var profileID int64
	err = s.db.QueryRowContext(ctx, `SELECT id FROM profiles WHERE username = ?`, "testuser").Scan(&profileID)
	s.Require().NoError(err)

	game := models.Game{
		ProfileID:      profileID,
		ChessComID:     "test123",
		PGN:            "test",
		TimeClass:      "blitz",
		Result:         "win",
		PlayedAs:       "white",
		Opponent:       "opp1",
		PlayedAt:       time.Now(),
		AnalysisStatus: "pending",
	}

	id, err := s.repo.Insert(ctx, game)
	s.Require().NoError(err)

	err = s.repo.UpdateStatus(ctx, id, "completed")
	s.Require().NoError(err)

	updated, err := s.repo.Get(ctx, id)
	s.Require().NoError(err)
	s.Assert().Equal("completed", updated.AnalysisStatus)
}

func (s *GameRepositorySuite) TestGamesNeedingAnalysis() {
	ctx := context.Background()

	// Create profile
	_, err := s.db.ExecContext(ctx, `INSERT INTO profiles (username) VALUES (?)`, "testuser")
	s.Require().NoError(err)

	var profileID int64
	err = s.db.QueryRowContext(ctx, `SELECT id FROM profiles WHERE username = ?`, "testuser").Scan(&profileID)
	s.Require().NoError(err)

	// Insert games with different statuses
	games := []models.Game{
		{
			ProfileID:      profileID,
			ChessComID:     "game1",
			PGN:            "test",
			TimeClass:      "blitz",
			Result:         "win",
			PlayedAs:       "white",
			Opponent:       "opp1",
			PlayedAt:       time.Now(),
			AnalysisStatus: "pending",
		},
		{
			ProfileID:      profileID,
			ChessComID:     "game2",
			PGN:            "test",
			TimeClass:      "rapid",
			Result:         "loss",
			PlayedAs:       "black",
			Opponent:       "opp2",
			PlayedAt:       time.Now(),
			AnalysisStatus: "completed",
		},
		{
			ProfileID:      profileID,
			ChessComID:     "game3",
			PGN:            "test",
			TimeClass:      "blitz",
			Result:         "draw",
			PlayedAs:       "white",
			Opponent:       "opp3",
			PlayedAt:       time.Now(),
			AnalysisStatus: "failed",
		},
	}

	_, err = s.repo.InsertBatch(ctx, games)
	s.Require().NoError(err)

	needing, err := s.repo.GamesNeedingAnalysis(ctx, profileID)
	s.Require().NoError(err)
	s.Assert().Len(needing, 2) // pending and failed, not completed
}

func TestGameRepositorySuite(t *testing.T) {
	suite.Run(t, new(GameRepositorySuite))
}
