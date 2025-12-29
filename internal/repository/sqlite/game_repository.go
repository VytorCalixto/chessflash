package sqlite

import (
	"context"
	"database/sql"
	"errors"

	"github.com/Masterminds/squirrel"
	"github.com/vytor/chessflash/internal/logger"
	"github.com/vytor/chessflash/internal/models"
	"github.com/vytor/chessflash/internal/repository"
)

var sqlBuilder = squirrel.StatementBuilder.PlaceholderFormat(squirrel.Question)

type gameRepository struct {
	db *sql.DB
}

// NewGameRepository creates a new GameRepository implementation
func NewGameRepository(db *sql.DB) repository.GameRepository {
	return &gameRepository{db: db}
}

func (r *gameRepository) Get(ctx context.Context, id int64) (*models.Game, error) {
	log := logger.FromContext(ctx).WithPrefix("game_repo")
	log.Debug("getting game: id=%d", id)

	var g models.Game
	err := r.db.QueryRowContext(ctx, `
SELECT id, profile_id, chess_com_id, pgn, time_class, result, played_as, opponent, player_rating, opponent_rating, played_at,
       eco_code, opening_name, opening_url, analysis_status, created_at
FROM games
WHERE id = ?
`, id).Scan(&g.ID, &g.ProfileID, &g.ChessComID, &g.PGN, &g.TimeClass, &g.Result, &g.PlayedAs, &g.Opponent, &g.PlayerRating, &g.OpponentRating, &g.PlayedAt, &g.ECOCode, &g.OpeningName, &g.OpeningURL, &g.AnalysisStatus, &g.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Debug("game not found: id=%d", id)
		} else {
			log.Error("failed to get game: %v", err)
		}
		return nil, err
	}
	log.Debug("game found: opponent=%s, result=%s", g.Opponent, g.Result)
	return &g, nil
}

func (r *gameRepository) List(ctx context.Context, filter models.GameFilter) ([]models.Game, error) {
	log := logger.FromContext(ctx).WithPrefix("game_repo")
	log.Debug("listing games with filter: profile_id=%d, time_class=%s, result=%s, opening=%s, opponent=%s",
		filter.ProfileID, filter.TimeClass, filter.Result, filter.OpeningName, filter.Opponent)

	query := sqlBuilder.Select(
		"id", "profile_id", "chess_com_id", "pgn", "time_class", "result", "played_as",
		"opponent", "player_rating", "opponent_rating", "played_at", "eco_code",
		"opening_name", "opening_url", "analysis_status", "created_at",
	).From("games")

	// Dynamic WHERE clauses
	if filter.ProfileID != 0 {
		query = query.Where(squirrel.Eq{"profile_id": filter.ProfileID})
	}
	if filter.TimeClass != "" {
		query = query.Where(squirrel.Eq{"time_class": filter.TimeClass})
	}
	if filter.Result != "" {
		query = query.Where(squirrel.Eq{"result": filter.Result})
	}
	if filter.OpeningName != "" {
		query = query.Where(squirrel.Eq{"opening_name": filter.OpeningName})
	}
	if filter.Opponent != "" {
		query = query.Where(squirrel.Eq{"opponent": filter.Opponent})
	}

	// Safe ORDER BY with validation
	orderBy := "played_at"
	if filter.OrderBy == "played_at" {
		orderBy = filter.OrderBy
	}
	orderDir := "DESC"
	if filter.OrderDir == "ASC" {
		orderDir = "ASC"
	} else if filter.OrderDir == "DESC" {
		orderDir = "DESC"
	}
	query = query.OrderBy(orderBy + " " + orderDir)

	// Pagination
	limit := filter.Limit
	if limit <= 0 {
		limit = 200
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}
	query = query.Limit(uint64(limit)).Offset(uint64(offset))

	sql, args, err := query.ToSql()
	if err != nil {
		log.Error("failed to build query: %v", err)
		return nil, err
	}

	rows, err := r.db.QueryContext(ctx, sql, args...)
	if err != nil {
		log.Error("failed to list games: %v", err)
		return nil, err
	}
	defer rows.Close()
	var games []models.Game
	for rows.Next() {
		var g models.Game
		if err := rows.Scan(&g.ID, &g.ProfileID, &g.ChessComID, &g.PGN, &g.TimeClass, &g.Result, &g.PlayedAs, &g.Opponent, &g.PlayerRating, &g.OpponentRating, &g.PlayedAt, &g.ECOCode, &g.OpeningName, &g.OpeningURL, &g.AnalysisStatus, &g.CreatedAt); err != nil {
			log.Error("failed to scan game row: %v", err)
			return nil, err
		}
		games = append(games, g)
	}
	log.Debug("found %d games", len(games))
	return games, rows.Err()
}

func (r *gameRepository) Count(ctx context.Context, filter models.GameFilter) (int, error) {
	log := logger.FromContext(ctx).WithPrefix("game_repo")
	log.Debug("counting games with filter: profile_id=%d, time_class=%s, result=%s, opening=%s, opponent=%s",
		filter.ProfileID, filter.TimeClass, filter.Result, filter.OpeningName, filter.Opponent)

	query := sqlBuilder.Select("COUNT(*)").From("games")

	// Same WHERE logic as List()
	if filter.ProfileID != 0 {
		query = query.Where(squirrel.Eq{"profile_id": filter.ProfileID})
	}
	if filter.TimeClass != "" {
		query = query.Where(squirrel.Eq{"time_class": filter.TimeClass})
	}
	if filter.Result != "" {
		query = query.Where(squirrel.Eq{"result": filter.Result})
	}
	if filter.OpeningName != "" {
		query = query.Where(squirrel.Eq{"opening_name": filter.OpeningName})
	}
	if filter.Opponent != "" {
		query = query.Where(squirrel.Eq{"opponent": filter.Opponent})
	}

	sql, args, err := query.ToSql()
	if err != nil {
		log.Error("failed to build query: %v", err)
		return 0, err
	}

	var count int
	err = r.db.QueryRowContext(ctx, sql, args...).Scan(&count)
	if err != nil {
		log.Error("failed to count games: %v", err)
		return 0, err
	}
	return count, nil
}

func (r *gameRepository) Insert(ctx context.Context, g models.Game) (int64, error) {
	log := logger.FromContext(ctx).WithPrefix("game_repo")
	log.Debug("inserting game: chess_com_id=%s, opponent=%s", g.ChessComID, g.Opponent)

	res, err := r.db.ExecContext(ctx, `
INSERT INTO games (
    profile_id, chess_com_id, pgn, time_class, result, played_as,
    opponent, player_rating, opponent_rating, played_at, eco_code, opening_name, opening_url, analysis_status
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(chess_com_id) DO UPDATE SET
    time_class = excluded.time_class,
    result = excluded.result,
    played_as = excluded.played_as,
    opponent = excluded.opponent,
    player_rating = excluded.player_rating,
    opponent_rating = excluded.opponent_rating,
    played_at = excluded.played_at,
    eco_code = excluded.eco_code,
    opening_name = excluded.opening_name,
    opening_url = excluded.opening_url
`, g.ProfileID, g.ChessComID, g.PGN, g.TimeClass, g.Result, g.PlayedAs, g.Opponent, g.PlayerRating, g.OpponentRating, g.PlayedAt, g.ECOCode, g.OpeningName, g.OpeningURL, g.AnalysisStatus)
	if err != nil {
		log.Error("failed to insert game: %v", err)
		return 0, err
	}
	if id, err := res.LastInsertId(); err == nil && id != 0 {
		log.Debug("game inserted: id=%d", id)
		return id, nil
	}
	var id int64
	err = r.db.QueryRowContext(ctx, `SELECT id FROM games WHERE chess_com_id = ?`, g.ChessComID).Scan(&id)
	if err != nil {
		log.Error("failed to get game id: %v", err)
	} else {
		log.Debug("game exists: id=%d", id)
	}
	return id, err
}

func (r *gameRepository) InsertBatch(ctx context.Context, games []models.Game) ([]int64, error) {
	log := logger.FromContext(ctx).WithPrefix("game_repo")
	log.Debug("batch inserting %d games", len(games))

	if len(games) == 0 {
		return nil, nil
	}

	var insertedIDs []int64
	err := tx(ctx, r.db, func(tx *sql.Tx) error {
		stmt, err := tx.PrepareContext(ctx, `
INSERT INTO games (
    profile_id, chess_com_id, pgn, time_class, result, played_as,
    opponent, player_rating, opponent_rating, played_at, eco_code, opening_name, opening_url, analysis_status
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(chess_com_id) DO NOTHING
`)
		if err != nil {
			log.Error("failed to prepare batch insert: %v", err)
			return err
		}
		defer stmt.Close()

		for _, g := range games {
			res, err := stmt.ExecContext(ctx, g.ProfileID, g.ChessComID, g.PGN, g.TimeClass, g.Result, g.PlayedAs, g.Opponent, g.PlayerRating, g.OpponentRating, g.PlayedAt, g.ECOCode, g.OpeningName, g.OpeningURL, g.AnalysisStatus)
			if err != nil {
				log.Error("failed to insert game chess_com_id=%s: %v", g.ChessComID, err)
				return err
			}
			if id, err := res.LastInsertId(); err == nil && id != 0 {
				insertedIDs = append(insertedIDs, id)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	log.Debug("batch insert completed, %d new games inserted", len(insertedIDs))
	return insertedIDs, nil
}

func (r *gameRepository) UpdateStatus(ctx context.Context, id int64, status string) error {
	log := logger.FromContext(ctx).WithPrefix("game_repo")
	log.Debug("updating game status: game_id=%d, status=%s", id, status)

	_, err := r.db.ExecContext(ctx, `UPDATE games SET analysis_status = ? WHERE id = ?`, status, id)
	if err != nil {
		log.Error("failed to update game status: %v", err)
	}
	return err
}

func (r *gameRepository) UpdateOpening(ctx context.Context, id int64, ecoCode, openingName string) error {
	log := logger.FromContext(ctx).WithPrefix("game_repo")
	log.Debug("updating game opening: game_id=%d, eco=%s, opening=%s", id, ecoCode, openingName)

	_, err := r.db.ExecContext(ctx, `
UPDATE games
SET eco_code = ?, opening_name = ?
WHERE id = ?
`, ecoCode, openingName, id)
	if err != nil {
		log.Error("failed to update game opening: %v", err)
	}
	return err
}

func (r *gameRepository) ResetProcessingToPending(ctx context.Context, profileID int64) error {
	log := logger.FromContext(ctx).WithPrefix("game_repo")
	log.Debug("resetting processing games to pending: profile_id=%d", profileID)

	_, err := r.db.ExecContext(ctx, `
UPDATE games
SET analysis_status = 'pending'
WHERE profile_id = ? AND analysis_status = 'processing'
`, profileID)
	if err != nil {
		log.Error("failed to reset processing games: %v", err)
	}
	return err
}

func (r *gameRepository) GamesNeedingAnalysis(ctx context.Context, profileID int64) ([]models.Game, error) {
	log := logger.FromContext(ctx).WithPrefix("game_repo")
	log.Debug("listing games needing analysis: profile_id=%d", profileID)

	rows, err := r.db.QueryContext(ctx, `
SELECT id, profile_id, chess_com_id, pgn, time_class, result, played_as, opponent, player_rating, opponent_rating, played_at,
       eco_code, opening_name, opening_url, analysis_status, created_at
FROM games
WHERE profile_id = ? AND analysis_status IN ('pending','processing','failed')
ORDER BY played_at DESC
`, profileID)
	if err != nil {
		log.Error("failed to list games needing analysis: %v", err)
		return nil, err
	}
	defer rows.Close()

	var games []models.Game
	for rows.Next() {
		var g models.Game
		if err := rows.Scan(&g.ID, &g.ProfileID, &g.ChessComID, &g.PGN, &g.TimeClass, &g.Result, &g.PlayedAs, &g.Opponent, &g.PlayerRating, &g.OpponentRating, &g.PlayedAt,
			&g.ECOCode, &g.OpeningName, &g.OpeningURL, &g.AnalysisStatus, &g.CreatedAt); err != nil {
			log.Error("failed to scan game row: %v", err)
			return nil, err
		}
		games = append(games, g)
	}
	log.Debug("found %d games needing analysis", len(games))
	return games, rows.Err()
}

func (r *gameRepository) CountGamesNeedingAnalysis(ctx context.Context, profileID int64) (int, error) {
	log := logger.FromContext(ctx).WithPrefix("game_repo")
	log.Debug("counting games needing analysis: profile_id=%d", profileID)

	var count int
	err := r.db.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM games
WHERE profile_id = ? AND analysis_status IN ('pending','processing','failed')
`, profileID).Scan(&count)
	if err != nil {
		log.Error("failed to count games needing analysis: %v", err)
		return 0, err
	}
	return count, nil
}

func (r *gameRepository) GetExistingChessComIDs(ctx context.Context, profileID int64) (map[string]bool, error) {
	log := logger.FromContext(ctx).WithPrefix("game_repo")
	log.Debug("loading existing chess_com_ids for profile_id=%d", profileID)

	rows, err := r.db.QueryContext(ctx, `SELECT chess_com_id FROM games WHERE profile_id = ?`, profileID)
	if err != nil {
		log.Error("failed to list chess_com_ids: %v", err)
		return nil, err
	}
	defer rows.Close()

	out := make(map[string]bool)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			log.Error("failed to scan chess_com_id: %v", err)
			return nil, err
		}
		out[id] = true
	}
	return out, rows.Err()
}
