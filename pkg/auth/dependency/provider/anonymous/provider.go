package anonymous

import (
	"github.com/sirupsen/logrus"
	"github.com/skygeario/skygear-server/pkg/core/db"
)

type providerImpl struct {
	sqlBuilder  db.SQLBuilder
	sqlExecutor db.SQLExecutor
	logger      *logrus.Entry
}

func newProvider(builder db.SQLBuilder, executor db.SQLExecutor, logger *logrus.Entry) *providerImpl {
	return &providerImpl{
		sqlBuilder:  builder,
		sqlExecutor: executor,
		logger:      logger,
	}
}

func NewProvider(builder db.SQLBuilder, executor db.SQLExecutor, logger *logrus.Entry) Provider {
	return newProvider(builder, executor, logger)
}

func (p providerImpl) CreatePrincipal(principal Principal) (err error) {
	// TODO: log

	// Create principal
	builder := p.sqlBuilder.Insert(p.sqlBuilder.FullTableName("principal")).Columns(
		"id",
		"provider",
		"user_id",
	).Values(
		principal.ID,
		providerAnonymous,
		principal.UserID,
	)

	_, err = p.sqlExecutor.ExecWith(builder)
	if err != nil {
		return
	}

	return
}

func (p providerImpl) DeletePrincipal(principalID string) (err error) {
	// TODO: log

	// Delete principal
	builder := p.sqlBuilder.Delete(p.sqlBuilder.FullTableName("principal")).
		Where("id = ?", principalID)

	_, err = p.sqlExecutor.ExecWith(builder)

	return
}
