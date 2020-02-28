package passwordhistory

import (
	"time"

	sq "github.com/Masterminds/squirrel"

	"github.com/skygeario/skygear-server/pkg/core/db"
	coreTime "github.com/skygeario/skygear-server/pkg/core/time"
	"github.com/skygeario/skygear-server/pkg/core/uuid"
)

type StoreImpl struct {
	sqlBuilder   db.SQLBuilder
	sqlExecutor  db.SQLExecutor
	timeProvider coreTime.Provider
}

func NewPasswordHistoryStore(builder db.SQLBuilder, executor db.SQLExecutor, timeProvider coreTime.Provider) *StoreImpl {
	return &StoreImpl{
		sqlBuilder:   builder,
		sqlExecutor:  executor,
		timeProvider: timeProvider,
	}
}

func (p *StoreImpl) CreatePasswordHistory(userID string, hashedPassword []byte, loggedAt time.Time) error {
	updateBuilder := p.insertPasswordHistoryBuilder(
		userID,
		hashedPassword,
		loggedAt,
	)
	if _, err := p.sqlExecutor.ExecWith(updateBuilder); err != nil {
		return err
	}
	return nil
}

func (p *StoreImpl) GetPasswordHistory(userID string, historySize, historyDays int) ([]PasswordHistory, error) {
	var err error
	var sizeHistory, daysHistory []PasswordHistory
	t := p.timeProvider.NowUTC()

	if historySize > 0 {
		sizeBuilder := p.basePasswordHistoryBuilder(userID).Limit(uint64(historySize))
		sizeHistory, err = p.doQueryPasswordHistory(sizeBuilder)
		if err != nil {
			return nil, err
		}
	}

	if historyDays > 0 {
		startOfDay := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
		since := startOfDay.AddDate(0, 0, -historyDays)
		daysBuilder := p.basePasswordHistoryBuilder(userID).
			Where("logged_at >= ?", since)
		daysHistory, err = p.doQueryPasswordHistory(daysBuilder)
		if err != nil {
			return nil, err
		}
	}

	if len(sizeHistory) > len(daysHistory) {
		return sizeHistory, nil
	}

	return daysHistory, nil
}

func (p *StoreImpl) RemovePasswordHistory(userID string, historySize, historyDays int) error {
	history, err := p.GetPasswordHistory(userID, historySize, historyDays)
	if err != nil {
		return err
	}

	if len(history) <= 0 {
		return nil
	}

	oldestTime := history[len(history)-1].LoggedAt
	ids := []interface{}{}
	for _, h := range history {
		ids = append(ids, h.ID)
	}

	builder := p.sqlBuilder.Tenant().
		Delete(p.sqlBuilder.FullTableName("password_history")).
		Where("user_id = ?", userID).
		Where("id NOT IN ("+sq.Placeholders(len(ids))+")", ids...).
		Where("logged_at < ?", oldestTime)

	_, err = p.sqlExecutor.ExecWith(builder)
	return err
}

func (p *StoreImpl) basePasswordHistoryBuilder(userID string) db.SelectBuilder {
	return p.sqlBuilder.Tenant().
		Select("id", "user_id", "password", "logged_at").
		From(p.sqlBuilder.FullTableName("password_history")).
		Where("user_id = ?", userID).
		OrderBy("logged_at DESC")
}

func (p *StoreImpl) insertPasswordHistoryBuilder(userID string, hashedPassword []byte, loggedAt time.Time) db.InsertBuilder {
	return p.sqlBuilder.Tenant().
		Insert(p.sqlBuilder.FullTableName("password_history")).
		Columns(
			"id",
			"user_id",
			"password",
			"logged_at",
		).
		Values(
			uuid.New(),
			userID,
			hashedPassword,
			loggedAt,
		)
}

func (p *StoreImpl) doQueryPasswordHistory(builder db.SelectBuilder) ([]PasswordHistory, error) {
	rows, err := p.sqlExecutor.QueryWith(builder)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []PasswordHistory{}
	for rows.Next() {
		var (
			id                string
			userID            string
			hashedPasswordStr string
			loggedAt          time.Time
		)
		if err := rows.Scan(&id, &userID, &hashedPasswordStr, &loggedAt); err != nil {
			return nil, err
		}
		passwordHistory := PasswordHistory{
			ID:             id,
			UserID:         userID,
			HashedPassword: []byte(hashedPasswordStr),
			LoggedAt:       loggedAt,
		}
		out = append(out, passwordHistory)
	}
	return out, nil
}

var (
	_ Store = &StoreImpl{}
)
