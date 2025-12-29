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

type FlashcardRepositorySuite struct {
	suite.Suite
	db   *sql.DB
	repo repository.FlashcardRepository
}

func (s *FlashcardRepositorySuite) SetupTest() {
	s.db = testutil.NewTestDB(s.T())
	s.repo = sqlite.NewFlashcardRepository(s.db)
}

func (s *FlashcardRepositorySuite) TearDownTest() {
	testutil.MustClose(s.T(), s.db)
}

func (s *FlashcardRepositorySuite) setupProfileAndGame() (int64, int64) {
	ctx := context.Background()

	// Create profile
	_, err := s.db.ExecContext(ctx, `INSERT INTO profiles (username) VALUES (?)`, "testuser")
	s.Require().NoError(err)

	var profileID int64
	err = s.db.QueryRowContext(ctx, `SELECT id FROM profiles WHERE username = ?`, "testuser").Scan(&profileID)
	s.Require().NoError(err)

	// Create game
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO games (profile_id, chess_com_id, pgn, time_class, result, played_as, opponent, played_at, analysis_status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, profileID, "game1", "test pgn", "blitz", "win", "white", "opponent1", time.Now(), "completed")
	s.Require().NoError(err)

	var gameID int64
	err = s.db.QueryRowContext(ctx, `SELECT id FROM games WHERE chess_com_id = ?`, "game1").Scan(&gameID)
	s.Require().NoError(err)

	return profileID, gameID
}

func (s *FlashcardRepositorySuite) TestInsertAndUpdate() {
	ctx := context.Background()
	_, gameID := s.setupProfileAndGame()

	// Create position
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO positions (game_id, move_number, fen, move_played, best_move, eval_before, eval_after, eval_diff, classification)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, gameID, 1, "rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1", "e2e4", "d2d4", 0.0, -50.0, -50.0, "mistake")
	s.Require().NoError(err)

	var positionID int64
	err = s.db.QueryRowContext(ctx, `SELECT id FROM positions WHERE game_id = ?`, gameID).Scan(&positionID)
	s.Require().NoError(err)

	flashcard := models.Flashcard{
		PositionID:    positionID,
		DueAt:         time.Now(),
		IntervalDays:  1,
		EaseFactor:    2.5,
		TimesReviewed: 0,
		TimesCorrect:  0,
	}

	id, err := s.repo.Insert(ctx, flashcard)
	s.Require().NoError(err)
	s.Assert().Greater(id, int64(0))

	// Update flashcard
	flashcard.ID = id
	flashcard.IntervalDays = 6
	flashcard.EaseFactor = 2.6
	flashcard.TimesReviewed = 1
	flashcard.TimesCorrect = 1
	flashcard.DueAt = time.Now().Add(6 * 24 * time.Hour)

	err = s.repo.Update(ctx, flashcard)
	s.Require().NoError(err)

	// Verify update
	var updatedInterval int
	var updatedEase float64
	err = s.db.QueryRowContext(ctx, `SELECT interval_days, ease_factor FROM flashcards WHERE id = ?`, id).Scan(&updatedInterval, &updatedEase)
	s.Require().NoError(err)
	s.Assert().Equal(6, updatedInterval)
	s.Assert().Equal(2.6, updatedEase)
}

func (s *FlashcardRepositorySuite) TestNextFlashcards() {
	ctx := context.Background()
	profileID, gameID := s.setupProfileAndGame()

	// Create positions
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO positions (game_id, move_number, fen, move_played, best_move, eval_before, eval_after, eval_diff, classification)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, gameID, 1, "fen1", "e2e4", "d2d4", 0.0, -50.0, -50.0, "mistake")
	s.Require().NoError(err)

	var positionID1 int64
	err = s.db.QueryRowContext(ctx, `SELECT id FROM positions WHERE move_number = 1 AND game_id = ?`, gameID).Scan(&positionID1)
	s.Require().NoError(err)

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO positions (game_id, move_number, fen, move_played, best_move, eval_before, eval_after, eval_diff, classification)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, gameID, 2, "fen2", "e7e5", "d7d5", 0.0, -100.0, -100.0, "blunder")
	s.Require().NoError(err)

	var positionID2 int64
	err = s.db.QueryRowContext(ctx, `SELECT id FROM positions WHERE move_number = 2 AND game_id = ?`, gameID).Scan(&positionID2)
	s.Require().NoError(err)

	// Create flashcards - one due now, one due later
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO flashcards (position_id, due_at, interval_days, ease_factor, times_reviewed, times_correct)
		VALUES (?, ?, ?, ?, ?, ?)
	`, positionID1, time.Now().Add(-1*time.Hour), 1, 2.5, 0, 0)
	s.Require().NoError(err)

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO flashcards (position_id, due_at, interval_days, ease_factor, times_reviewed, times_correct)
		VALUES (?, ?, ?, ?, ?, ?)
	`, positionID2, time.Now().Add(24*time.Hour), 1, 2.5, 0, 0)
	s.Require().NoError(err)

	cards, err := s.repo.NextFlashcards(ctx, profileID, 10)
	s.Require().NoError(err)
	s.Assert().Len(cards, 1) // Only the due one
	s.Assert().Equal(positionID1, cards[0].PositionID)
}

func (s *FlashcardRepositorySuite) TestFlashcardWithPosition() {
	ctx := context.Background()
	profileID, gameID := s.setupProfileAndGame()

	// Create position
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO positions (game_id, move_number, fen, move_played, best_move, eval_before, eval_after, eval_diff, classification)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, gameID, 1, "rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1", "e2e4", "d2d4", 0.0, -50.0, -50.0, "mistake")
	s.Require().NoError(err)

	var positionID int64
	err = s.db.QueryRowContext(ctx, `SELECT id FROM positions WHERE game_id = ?`, gameID).Scan(&positionID)
	s.Require().NoError(err)

	// Create flashcard
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO flashcards (position_id, due_at, interval_days, ease_factor, times_reviewed, times_correct)
		VALUES (?, ?, ?, ?, ?, ?)
	`, positionID, time.Now(), 1, 2.5, 0, 0)
	s.Require().NoError(err)

	var flashcardID int64
	err = s.db.QueryRowContext(ctx, `SELECT id FROM flashcards WHERE position_id = ?`, positionID).Scan(&flashcardID)
	s.Require().NoError(err)

	fp, err := s.repo.FlashcardWithPosition(ctx, flashcardID, profileID)
	s.Require().NoError(err)
	s.Require().NotNil(fp)
	s.Assert().Equal(positionID, fp.PositionID)
	s.Assert().Equal(gameID, fp.GameID)
	s.Assert().Equal("mistake", fp.Classification)
	s.Assert().Equal("e2e4", fp.MovePlayed)
}

func (s *FlashcardRepositorySuite) TestInsertReviewHistory() {
	ctx := context.Background()
	_, gameID := s.setupProfileAndGame()

	// Create position and flashcard
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO positions (game_id, move_number, fen, move_played, best_move, eval_before, eval_after, eval_diff, classification)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, gameID, 1, "fen1", "e2e4", "d2d4", 0.0, -50.0, -50.0, "mistake")
	s.Require().NoError(err)

	var positionID int64
	err = s.db.QueryRowContext(ctx, `SELECT id FROM positions WHERE game_id = ?`, gameID).Scan(&positionID)
	s.Require().NoError(err)

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO flashcards (position_id, due_at, interval_days, ease_factor, times_reviewed, times_correct)
		VALUES (?, ?, ?, ?, ?, ?)
	`, positionID, time.Now(), 1, 2.5, 0, 0)
	s.Require().NoError(err)

	var flashcardID int64
	err = s.db.QueryRowContext(ctx, `SELECT id FROM flashcards WHERE position_id = ?`, positionID).Scan(&flashcardID)
	s.Require().NoError(err)

	err = s.repo.InsertReviewHistory(ctx, flashcardID, 2, 5.5)
	s.Require().NoError(err)

	// Verify history was inserted
	var quality int
	var timeSeconds float64
	err = s.db.QueryRowContext(ctx, `SELECT quality, time_seconds FROM review_history WHERE flashcard_id = ?`, flashcardID).Scan(&quality, &timeSeconds)
	s.Require().NoError(err)
	s.Assert().Equal(2, quality)
	s.Assert().Equal(5.5, timeSeconds)
}

func TestFlashcardRepositorySuite(t *testing.T) {
	suite.Run(t, new(FlashcardRepositorySuite))
}
