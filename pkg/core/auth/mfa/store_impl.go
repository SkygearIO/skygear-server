package mfa

import (
	"database/sql"
	"sort"
	gotime "time"

	"github.com/lib/pq"

	coreAuth "github.com/skygeario/skygear-server/pkg/core/auth"
	"github.com/skygeario/skygear-server/pkg/core/config"
	"github.com/skygeario/skygear-server/pkg/core/db"
	"github.com/skygeario/skygear-server/pkg/core/time"
	"github.com/skygeario/skygear-server/pkg/core/uuid"
)

type StoreImpl struct {
	mfaConfig    *config.MFAConfiguration
	sqlBuilder   db.SQLBuilder
	sqlExecutor  db.SQLExecutor
	timeProvider time.Provider
}

func NewStore(
	mfaConfig *config.MFAConfiguration,
	sqlBuilder db.SQLBuilder,
	sqlExecutor db.SQLExecutor,
	timeProvider time.Provider,
) *StoreImpl {
	return &StoreImpl{
		mfaConfig:    mfaConfig,
		sqlBuilder:   sqlBuilder,
		sqlExecutor:  sqlExecutor,
		timeProvider: timeProvider,
	}
}

func sortRecoveryCodeAuthenticatorSlice(s []RecoveryCodeAuthenticator) {
	sort.Slice(s, func(i, j int) bool {
		a := s[i]
		b := s[j]
		return a.Code < b.Code
	})
}

func sortAuthenticatorSlice(s []Authenticator) {
	sort.Slice(s, func(i, j int) bool {
		a := s[i]
		b := s[j]
		return a.GetActivatedAt().After(*b.GetActivatedAt())
	})
}

func (s *StoreImpl) scanTOTPAuthenticator(scanner db.Scanner, a *TOTPAuthenticator) error {
	var activatedAt pq.NullTime
	err := scanner.Scan(
		&a.ID,
		&a.UserID,
		&a.Type,
		&a.Activated,
		&a.CreatedAt,
		&activatedAt,
		&a.Secret,
		&a.DisplayName,
	)
	if err != nil {
		return err
	}
	if activatedAt.Valid {
		a.ActivatedAt = &activatedAt.Time
	}
	return nil
}

func (s *StoreImpl) scanOOBAuthenticator(scanner db.Scanner, a *OOBAuthenticator) error {
	var activatedAt pq.NullTime
	err := scanner.Scan(
		&a.ID,
		&a.UserID,
		&a.Type,
		&a.Activated,
		&a.CreatedAt,
		&activatedAt,
		&a.Channel,
		&a.Phone,
		&a.Email,
	)
	if err != nil {
		return err
	}
	if activatedAt.Valid {
		a.ActivatedAt = &activatedAt.Time
	}
	return nil
}

func (s *StoreImpl) GetRecoveryCode(userID string) (output []RecoveryCodeAuthenticator, err error) {
	builder := s.sqlBuilder.Tenant().
		Select(
			"a.id",
			"a.user_id",
			"a.type",
			"arc.code",
			"arc.created_at",
			"arc.consumed",
		).
		From(s.sqlBuilder.FullTableName("authenticator"), "a").
		Join(
			s.sqlBuilder.FullTableName("authenticator_recovery_code"),
			"arc",
			"a.id = arc.id",
		).
		Where("a.user_id = ?", userID)
	rows, err := s.sqlExecutor.QueryWith(builder)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var a RecoveryCodeAuthenticator
		err = rows.Scan(
			&a.ID,
			&a.UserID,
			&a.Type,
			&a.Code,
			&a.CreatedAt,
			&a.Consumed,
		)
		if err != nil {
			return
		}
		output = append(output, a)
	}

	sortRecoveryCodeAuthenticatorSlice(output)

	return
}

func (s *StoreImpl) DeleteRecoveryCode(userID string) error {
	old, err := s.GetRecoveryCode(userID)
	if err != nil {
		return err
	}
	var ids []string
	for _, a := range old {
		ids = append(ids, a.ID)
	}

	if len(ids) > 0 {
		q1 := s.sqlBuilder.Tenant().
			Delete(s.sqlBuilder.FullTableName("authenticator_recovery_code")).
			Where("id = ANY (?)", pq.Array(ids))

		_, err = s.sqlExecutor.ExecWith(q1)
		if err != nil {
			return err
		}

		q2 := s.sqlBuilder.Tenant().
			Delete(s.sqlBuilder.FullTableName("authenticator")).
			Where("id = ANY (?)", pq.Array(ids))

		_, err = s.sqlExecutor.ExecWith(q2)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *StoreImpl) GenerateRecoveryCode(userID string) ([]RecoveryCodeAuthenticator, error) {
	err := s.DeleteRecoveryCode(userID)
	if err != nil {
		return nil, err
	}

	q1 := s.sqlBuilder.Tenant().
		Insert(s.sqlBuilder.FullTableName("authenticator")).
		Columns(
			"id",
			"type",
			"user_id",
		)
	q2 := s.sqlBuilder.Tenant().
		Insert(s.sqlBuilder.FullTableName("authenticator_recovery_code")).
		Columns(
			"id",
			"code",
			"created_at",
			"consumed",
		)

	now := s.timeProvider.NowUTC()
	var output []RecoveryCodeAuthenticator
	for i := 0; i < s.mfaConfig.RecoveryCode.Count; i++ {
		a := RecoveryCodeAuthenticator{
			ID:        uuid.New(),
			UserID:    userID,
			Type:      coreAuth.AuthenticatorTypeRecoveryCode,
			Code:      GenerateRandomRecoveryCode(),
			CreatedAt: now,
			Consumed:  false,
		}
		q1 = q1.Values(
			a.ID,
			a.Type,
			a.UserID,
		)
		q2 = q2.Values(
			a.ID,
			a.Code,
			a.CreatedAt,
			a.Consumed,
		)
		output = append(output, a)
	}

	_, err = s.sqlExecutor.ExecWith(q1)
	if err != nil {
		return nil, err
	}
	_, err = s.sqlExecutor.ExecWith(q2)
	if err != nil {
		return nil, err
	}

	sortRecoveryCodeAuthenticatorSlice(output)

	return output, nil
}

func (s *StoreImpl) UpdateRecoveryCode(a *RecoveryCodeAuthenticator) error {
	q1 := s.sqlBuilder.Tenant().
		Update(s.sqlBuilder.FullTableName("authenticator_recovery_code")).
		Set("consumed", a.Consumed).
		Where("id = ?", a.ID)
	_, err := s.sqlExecutor.ExecWith(q1)
	if err != nil {
		return err
	}
	return err
}

func (s *StoreImpl) DeleteAllBearerToken(userID string) error {
	q1 := s.sqlBuilder.Tenant().
		Select("a.id").
		From(s.sqlBuilder.FullTableName("authenticator"), "a").
		Where("a.user_id = ? AND a.type = ?", userID, coreAuth.AuthenticatorTypeBearerToken)

	rows1, err := s.sqlExecutor.QueryWith(q1)
	if err != nil {
		return err
	}
	defer rows1.Close()

	var ids []string
	for rows1.Next() {
		var id string
		err = rows1.Scan(&id)
		if err != nil {
			return err
		}
		ids = append(ids, id)
	}

	return s.deleteBearerTokenByIDs(ids)
}

func (s *StoreImpl) DeleteExpiredBearerToken(userID string) error {
	now := s.timeProvider.NowUTC()
	q1 := s.sqlBuilder.Tenant().
		Select("a.id").
		From(s.sqlBuilder.FullTableName("authenticator"), "a").
		Join(
			s.sqlBuilder.FullTableName("authenticator_bearer_token"),
			"abt",
			"a.id = abt.id",
		).
		Where("a.user_id = ? AND abt.expire_at < ?", userID, now)

	rows1, err := s.sqlExecutor.QueryWith(q1)
	if err != nil {
		return err
	}
	defer rows1.Close()

	var ids []string
	for rows1.Next() {
		var id string
		err = rows1.Scan(&id)
		if err != nil {
			return err
		}
		ids = append(ids, id)
	}

	return s.deleteBearerTokenByIDs(ids)
}

func (s *StoreImpl) deleteBearerTokenByParentIDs(parentIDs []string) error {
	if len(parentIDs) <= 0 {
		return nil
	}

	q1 := s.sqlBuilder.Tenant().
		Select(
			"a.id",
		).
		From(s.sqlBuilder.FullTableName("authenticator"), "a").
		Join(
			s.sqlBuilder.FullTableName("authenticator_bearer_token"),
			"abt",
			"a.id = abt.id",
		).
		Where("abt.parent_id = ANY (?)", pq.Array(parentIDs))

	rows1, err := s.sqlExecutor.QueryWith(q1)
	if err != nil {
		return err
	}
	defer rows1.Close()

	var ids []string
	for rows1.Next() {
		var id string
		err = rows1.Scan(&id)
		if err != nil {
			return err
		}
		ids = append(ids, id)
	}

	return s.deleteBearerTokenByIDs(ids)
}

func (s *StoreImpl) deleteBearerTokenByIDs(ids []string) error {
	if len(ids) <= 0 {
		return nil
	}
	q2 := s.sqlBuilder.Tenant().
		Delete(s.sqlBuilder.FullTableName("authenticator_bearer_token")).
		Where("id = ANY (?)", pq.Array(ids))
	_, err := s.sqlExecutor.ExecWith(q2)
	if err != nil {
		return err
	}

	q3 := s.sqlBuilder.Tenant().
		Delete(s.sqlBuilder.FullTableName("authenticator")).
		Where("id = ANY (?)", pq.Array(ids))

	_, err = s.sqlExecutor.ExecWith(q3)
	if err != nil {
		return err
	}
	return nil
}

func (s *StoreImpl) CreateBearerToken(a *BearerTokenAuthenticator) error {
	q1 := s.sqlBuilder.Tenant().
		Insert(s.sqlBuilder.FullTableName("authenticator")).
		Columns(
			"id",
			"type",
			"user_id",
		).
		Values(
			a.ID,
			a.Type,
			a.UserID,
		)
	_, err := s.sqlExecutor.ExecWith(q1)
	if err != nil {
		return err
	}

	q2 := s.sqlBuilder.Tenant().
		Insert(s.sqlBuilder.FullTableName("authenticator_bearer_token")).
		Columns(
			"id",
			"parent_id",
			"token",
			"created_at",
			"expire_at",
		).
		Values(
			a.ID,
			a.ParentID,
			a.Token,
			a.CreatedAt,
			a.ExpireAt,
		)
	_, err = s.sqlExecutor.ExecWith(q2)
	if err != nil {
		return err
	}

	return nil
}

func (s *StoreImpl) GetBearerTokenByToken(userID string, token string) (*BearerTokenAuthenticator, error) {
	q1 := s.sqlBuilder.Tenant().
		Select(
			"a.id",
			"a.user_id",
			"a.type",
			"abt.parent_id",
			"abt.token",
			"abt.created_at",
			"abt.expire_at",
		).
		From(s.sqlBuilder.FullTableName("authenticator"), "a").
		Join(
			s.sqlBuilder.FullTableName("authenticator_bearer_token"),
			"abt",
			"a.id = abt.id",
		).
		// SECURITY(louis): Ideally we should compare the bearer token in constant time.
		// However, it requires us to fetch all bearer tokens. The number can be unbound
		// because we do not limit the number of the bearer tokens.
		Where("a.user_id = ? AND abt.token = ?", userID, token)

	row, err := s.sqlExecutor.QueryRowWith(q1)
	if err != nil {
		return nil, err
	}

	var a BearerTokenAuthenticator
	err = row.Scan(
		&a.ID,
		&a.UserID,
		&a.Type,
		&a.ParentID,
		&a.Token,
		&a.CreatedAt,
		&a.ExpireAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			err = ErrNoAuthenticators
		}
		return nil, err
	}
	return &a, nil
}

func (s *StoreImpl) ListAuthenticators(userID string) ([]Authenticator, error) {
	q1 := s.sqlBuilder.Tenant().
		Select(
			"a.id",
			"a.user_id",
			"a.type",
			"at.activated",
			"at.created_at",
			"at.activated_at",
			"at.secret",
			"at.display_name",
		).
		From(s.sqlBuilder.FullTableName("authenticator"), "a").
		Join(
			s.sqlBuilder.FullTableName("authenticator_totp"),
			"at",
			"a.id = at.id",
		).
		Where("a.user_id = ? AND at.activated = TRUE", userID)
	rows1, err := s.sqlExecutor.QueryWith(q1)
	if err != nil {
		return nil, err
	}
	defer rows1.Close()

	var totps []TOTPAuthenticator
	for rows1.Next() {
		var a TOTPAuthenticator
		err = s.scanTOTPAuthenticator(rows1, &a)
		if err != nil {
			return nil, err
		}
		totps = append(totps, a)
	}

	q2 := s.sqlBuilder.Tenant().
		Select(
			"a.id",
			"a.user_id",
			"a.type",
			"ao.activated",
			"ao.created_at",
			"ao.activated_at",
			"ao.channel",
			"ao.phone",
			"ao.email",
		).
		From(s.sqlBuilder.FullTableName("authenticator"), "a").
		Join(
			s.sqlBuilder.FullTableName("authenticator_oob"),
			"ao",
			"a.id = ao.id",
		).
		Where("a.user_id = ? AND ao.activated = TRUE", userID)
	rows2, err := s.sqlExecutor.QueryWith(q2)
	if err != nil {
		return nil, err
	}
	defer rows2.Close()

	var oobs []OOBAuthenticator
	for rows2.Next() {
		var a OOBAuthenticator
		err = s.scanOOBAuthenticator(rows2, &a)
		if err != nil {
			return nil, err
		}
		oobs = append(oobs, a)
	}

	output := []Authenticator{}
	for _, a := range totps {
		output = append(output, a)
	}
	for _, a := range oobs {
		output = append(output, a)
	}

	sortAuthenticatorSlice(output)

	return output, nil
}

func (s *StoreImpl) CreateTOTP(a *TOTPAuthenticator) error {
	q1 := s.sqlBuilder.Tenant().
		Insert(s.sqlBuilder.FullTableName("authenticator")).
		Columns(
			"id",
			"type",
			"user_id",
		).
		Values(
			a.ID,
			a.Type,
			a.UserID,
		)
	_, err := s.sqlExecutor.ExecWith(q1)
	if err != nil {
		return err
	}

	q2 := s.sqlBuilder.Tenant().
		Insert(s.sqlBuilder.FullTableName("authenticator_totp")).
		Columns(
			"id",
			"activated",
			"created_at",
			"activated_at",
			"secret",
			"display_name",
		).
		Values(
			a.ID,
			a.Activated,
			a.CreatedAt,
			a.ActivatedAt,
			a.Secret,
			a.DisplayName,
		)
	_, err = s.sqlExecutor.ExecWith(q2)
	if err != nil {
		return err
	}

	return nil
}

func (s *StoreImpl) GetTOTP(userID string, id string) (*TOTPAuthenticator, error) {
	q1 := s.sqlBuilder.Tenant().
		Select(
			"a.id",
			"a.user_id",
			"a.type",
			"at.activated",
			"at.created_at",
			"at.activated_at",
			"at.secret",
			"at.display_name",
		).
		From(s.sqlBuilder.FullTableName("authenticator"), "a").
		Join(
			s.sqlBuilder.FullTableName("authenticator_totp"),
			"at",
			"a.id = at.id",
		).
		Where("a.user_id = ? AND a.id = ?", userID, id)

	row, err := s.sqlExecutor.QueryRowWith(q1)
	if err != nil {
		return nil, err
	}

	var a TOTPAuthenticator
	err = s.scanTOTPAuthenticator(row, &a)
	if err != nil {
		if err == sql.ErrNoRows {
			err = ErrNoAuthenticators
		}
		return nil, err
	}
	return &a, nil
}

func (s *StoreImpl) UpdateTOTP(a *TOTPAuthenticator) error {
	q1 := s.sqlBuilder.Tenant().
		Update(s.sqlBuilder.FullTableName("authenticator_totp")).
		Set("activated", a.Activated).
		Set("activated_at", a.ActivatedAt).
		Where("id = ?", a.ID)
	_, err := s.sqlExecutor.ExecWith(q1)
	if err != nil {
		return err
	}
	return err
}

func (s *StoreImpl) DeleteTOTP(a *TOTPAuthenticator) error {
	return s.deleteTOTPByIDs([]string{a.ID})
}

func (s *StoreImpl) deleteTOTPByIDs(ids []string) error {
	if len(ids) <= 0 {
		return nil
	}

	err := s.deleteBearerTokenByParentIDs(ids)
	if err != nil {
		return err
	}

	q2 := s.sqlBuilder.Tenant().
		Delete(s.sqlBuilder.FullTableName("authenticator_totp")).
		Where("id = ANY (?)", pq.Array(ids))
	r2, err := s.sqlExecutor.ExecWith(q2)
	if err != nil {
		return err
	}
	count, err := r2.RowsAffected()
	if err != nil {
		return err
	}
	if int(count) != len(ids) {
		return ErrNoAuthenticators
	}

	q3 := s.sqlBuilder.Tenant().
		Delete(s.sqlBuilder.FullTableName("authenticator")).
		Where("id = ANY (?)", pq.Array(ids))

	r3, err := s.sqlExecutor.ExecWith(q3)
	if err != nil {
		return err
	}
	count, err = r3.RowsAffected()
	if err != nil {
		return err
	}
	if int(count) != len(ids) {
		return ErrNoAuthenticators
	}

	return nil
}

func (s *StoreImpl) DeleteInactiveTOTP(userID string) error {
	q1 := s.sqlBuilder.Tenant().
		Select("a.id").
		From(s.sqlBuilder.FullTableName("authenticator"), "a").
		Join(
			s.sqlBuilder.FullTableName("authenticator_totp"),
			"at",
			"a.id = at.id",
		).
		Where("a.user_id = ? AND at.activated = FALSE", userID)

	rows1, err := s.sqlExecutor.QueryWith(q1)
	if err != nil {
		return err
	}
	defer rows1.Close()

	var ids []string
	for rows1.Next() {
		var id string
		err = rows1.Scan(&id)
		if err != nil {
			return err
		}
		ids = append(ids, id)
	}

	return s.deleteTOTPByIDs(ids)
}

func (s *StoreImpl) GetOnlyInactiveTOTP(userID string) (*TOTPAuthenticator, error) {
	q1 := s.sqlBuilder.Tenant().
		Select(
			"a.id",
			"a.user_id",
			"a.type",
			"at.activated",
			"at.created_at",
			"at.activated_at",
			"at.secret",
			"at.display_name",
		).
		From(s.sqlBuilder.FullTableName("authenticator"), "a").
		Join(
			s.sqlBuilder.FullTableName("authenticator_totp"),
			"at",
			"a.id = at.id",
		).
		Where("a.user_id = ? AND at.activated = FALSE", userID)

	row, err := s.sqlExecutor.QueryRowWith(q1)
	if err != nil {
		return nil, err
	}

	var a TOTPAuthenticator
	err = s.scanTOTPAuthenticator(row, &a)
	if err != nil {
		if err == sql.ErrNoRows {
			err = ErrNoAuthenticators
		}
		return nil, err
	}
	return &a, nil
}

func (s *StoreImpl) CreateOOB(a *OOBAuthenticator) error {
	q1 := s.sqlBuilder.Tenant().
		Insert(s.sqlBuilder.FullTableName("authenticator")).
		Columns(
			"id",
			"type",
			"user_id",
		).
		Values(
			a.ID,
			a.Type,
			a.UserID,
		)
	_, err := s.sqlExecutor.ExecWith(q1)
	if err != nil {
		return err
	}

	q2 := s.sqlBuilder.Tenant().
		Insert(s.sqlBuilder.FullTableName("authenticator_oob")).
		Columns(
			"id",
			"activated",
			"created_at",
			"activated_at",
			"channel",
			"phone",
			"email",
		).
		Values(
			a.ID,
			a.Activated,
			a.CreatedAt,
			a.ActivatedAt,
			a.Channel,
			a.Phone,
			a.Email,
		)
	_, err = s.sqlExecutor.ExecWith(q2)
	if err != nil {
		return err
	}

	return nil
}

func (s *StoreImpl) GetOOB(userID string, id string) (*OOBAuthenticator, error) {
	q1 := s.sqlBuilder.Tenant().
		Select(
			"a.id",
			"a.user_id",
			"a.type",
			"ao.activated",
			"ao.created_at",
			"ao.activated_at",
			"ao.channel",
			"ao.phone",
			"ao.email",
		).
		From(s.sqlBuilder.FullTableName("authenticator"), "a").
		Join(
			s.sqlBuilder.FullTableName("authenticator_oob"),
			"ao",
			"a.id = ao.id",
		).
		Where("a.user_id = ? AND a.id = ?", userID, id)

	row, err := s.sqlExecutor.QueryRowWith(q1)
	if err != nil {
		return nil, err
	}

	var a OOBAuthenticator
	err = s.scanOOBAuthenticator(row, &a)
	if err != nil {
		if err == sql.ErrNoRows {
			err = ErrNoAuthenticators
		}
		return nil, err
	}
	return &a, nil
}

func (s *StoreImpl) GetOnlyInactiveOOB(userID string) (*OOBAuthenticator, error) {
	q1 := s.sqlBuilder.Tenant().
		Select(
			"a.id",
			"a.user_id",
			"a.type",
			"ao.activated",
			"ao.created_at",
			"ao.activated_at",
			"ao.channel",
			"ao.phone",
			"ao.email",
		).
		From(s.sqlBuilder.FullTableName("authenticator"), "a").
		Join(
			s.sqlBuilder.FullTableName("authenticator_oob"),
			"ao",
			"a.id = ao.id",
		).
		Where("a.user_id = ? AND ao.activated = FALSE", userID)

	row, err := s.sqlExecutor.QueryRowWith(q1)
	if err != nil {
		return nil, err
	}

	var a OOBAuthenticator
	err = s.scanOOBAuthenticator(row, &a)
	if err != nil {
		if err == sql.ErrNoRows {
			err = ErrNoAuthenticators
		}
		return nil, err
	}
	return &a, nil
}

func (s *StoreImpl) GetOOBByChannel(userID string, channel coreAuth.AuthenticatorOOBChannel, phone string, email string) (*OOBAuthenticator, error) {
	q1 := s.sqlBuilder.Tenant().
		Select(
			"a.id",
			"a.user_id",
			"a.type",
			"ao.activated",
			"ao.created_at",
			"ao.activated_at",
			"ao.channel",
			"ao.phone",
			"ao.email",
		).
		From(s.sqlBuilder.FullTableName("authenticator"), "a").
		Join(
			s.sqlBuilder.FullTableName("authenticator_oob"),
			"ao",
			"a.id = ao.id",
		)
	switch channel {
	case coreAuth.AuthenticatorOOBChannelSMS:
		q1 = q1.Where("a.user_id = ? AND ao.channel = ? AND ao.phone = ?", userID, channel, phone)
	case coreAuth.AuthenticatorOOBChannelEmail:
		q1 = q1.Where("a.user_id = ? AND ao.channel = ? AND ao.email = ?", userID, channel, email)
	default:
		panic("mfa: unknown authenticator channel")
	}

	row, err := s.sqlExecutor.QueryRowWith(q1)
	if err != nil {
		return nil, err
	}

	var a OOBAuthenticator
	err = s.scanOOBAuthenticator(row, &a)
	if err != nil {
		if err == sql.ErrNoRows {
			err = ErrNoAuthenticators
		}
		return nil, err
	}
	return &a, nil
}

func (s *StoreImpl) UpdateOOB(a *OOBAuthenticator) error {
	q1 := s.sqlBuilder.Tenant().
		Update(s.sqlBuilder.FullTableName("authenticator_oob")).
		Set("activated", a.Activated).
		Set("activated_at", a.ActivatedAt).
		Where("id = ?", a.ID)
	_, err := s.sqlExecutor.ExecWith(q1)
	if err != nil {
		return err
	}
	return err
}

func (s *StoreImpl) DeleteOOB(a *OOBAuthenticator) error {
	return s.deleteOOBByIDs([]string{a.ID})
}

func (s *StoreImpl) DeleteInactiveOOB(userID string, exceptID string) error {
	q1 := s.sqlBuilder.Tenant().
		Select("a.id").
		From(s.sqlBuilder.FullTableName("authenticator"), "a").
		Join(
			s.sqlBuilder.FullTableName("authenticator_oob"),
			"ao",
			"a.id = ao.id",
		).
		Where("a.user_id = ? AND ao.activated = FALSE", userID)

	rows1, err := s.sqlExecutor.QueryWith(q1)
	if err != nil {
		return err
	}
	defer rows1.Close()

	var ids []string
	for rows1.Next() {
		var id string
		err = rows1.Scan(&id)
		if err != nil {
			return err
		}
		if id != exceptID {
			ids = append(ids, id)
		}
	}

	return s.deleteOOBByIDs(ids)
}

func (s *StoreImpl) deleteOOBByIDs(ids []string) error {
	if len(ids) <= 0 {
		return nil
	}

	err := s.deleteBearerTokenByParentIDs(ids)
	if err != nil {
		return err
	}

	err = s.deleteOOBCodeByAuthenticatorIDs(ids)
	if err != nil {
		return err
	}

	q1 := s.sqlBuilder.Tenant().
		Delete(s.sqlBuilder.FullTableName("authenticator_oob")).
		Where("id = ANY (?)", pq.Array(ids))
	r1, err := s.sqlExecutor.ExecWith(q1)
	if err != nil {
		return err
	}
	count, err := r1.RowsAffected()
	if err != nil {
		return err
	}
	if int(count) != len(ids) {
		return ErrNoAuthenticators
	}

	q2 := s.sqlBuilder.Tenant().
		Delete(s.sqlBuilder.FullTableName("authenticator")).
		Where("id = ANY (?)", pq.Array(ids))
	r2, err := s.sqlExecutor.ExecWith(q2)
	if err != nil {
		return err
	}
	count, err = r2.RowsAffected()
	if err != nil {
		return err
	}
	if int(count) != len(ids) {
		return ErrNoAuthenticators
	}

	return nil
}

func (s *StoreImpl) GetValidOOBCode(userID string, t gotime.Time) ([]OOBCode, error) {
	q1 := s.sqlBuilder.Tenant().
		Select(
			"aoc.id",
			"a.user_id",
			"aoc.authenticator_id",
			"aoc.code",
			"aoc.created_at",
			"aoc.expire_at",
		).
		From(s.sqlBuilder.FullTableName("authenticator_oob_code"), "aoc").
		Join(
			s.sqlBuilder.FullTableName("authenticator"),
			"a",
			"a.id = aoc.authenticator_id",
		).
		Where("a.user_id = ? AND aoc.expire_at > ?", userID, t)
	rows, err := s.sqlExecutor.QueryWith(q1)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var output []OOBCode
	for rows.Next() {
		var a OOBCode
		err = rows.Scan(
			&a.ID,
			&a.UserID,
			&a.AuthenticatorID,
			&a.Code,
			&a.CreatedAt,
			&a.ExpireAt,
		)
		if err != nil {
			return nil, err
		}
		output = append(output, a)
	}

	return output, nil
}

func (s *StoreImpl) CreateOOBCode(c *OOBCode) error {
	q1 := s.sqlBuilder.Tenant().
		Insert(s.sqlBuilder.FullTableName("authenticator_oob_code")).
		Columns(
			"id",
			"authenticator_id",
			"code",
			"created_at",
			"expire_at",
		).
		Values(
			c.ID,
			c.AuthenticatorID,
			c.Code,
			c.CreatedAt,
			c.ExpireAt,
		)
	_, err := s.sqlExecutor.ExecWith(q1)
	if err != nil {
		return err
	}
	return nil
}

func (s *StoreImpl) DeleteOOBCode(c *OOBCode) error {
	q1 := s.sqlBuilder.Tenant().
		Delete(s.sqlBuilder.FullTableName("authenticator_oob_code")).
		Where("id = ?", c.ID)
	_, err := s.sqlExecutor.ExecWith(q1)
	if err != nil {
		return err
	}
	return nil
}

func (s *StoreImpl) deleteOOBCodeByAuthenticatorIDs(authenticatorIDs []string) error {
	if len(authenticatorIDs) <= 0 {
		return nil
	}

	q1 := s.sqlBuilder.Tenant().
		Delete(s.sqlBuilder.FullTableName("authenticator_oob_code")).
		Where("authenticator_id = ANY (?)", pq.Array(authenticatorIDs))
	_, err := s.sqlExecutor.ExecWith(q1)
	if err != nil {
		return err
	}

	return nil
}

var (
	_ Store = &StoreImpl{}
)
