package passwordhistory

import (
	"github.com/sirupsen/logrus"

	"github.com/skygeario/skygear-server/pkg/core/logging"
)

func NewPwHousekeeper(
	passwordHistoryStore Store,
	loggerFactory logging.Factory,
	pwHistorySize int,
	pwHistoryDays int,
	passwordHistoryEnabled bool,
) *PwHousekeeper {
	return &PwHousekeeper{
		passwordHistoryStore:   passwordHistoryStore,
		logger:                 loggerFactory.NewLogger("password-housekeeper"),
		pwHistorySize:          pwHistorySize,
		pwHistoryDays:          pwHistoryDays,
		passwordHistoryEnabled: passwordHistoryEnabled,
	}
}

type PwHousekeeper struct {
	passwordHistoryStore   Store
	logger                 *logrus.Entry
	pwHistorySize          int
	pwHistoryDays          int
	passwordHistoryEnabled bool
}

func (p *PwHousekeeper) Housekeep(authID string) (err error) {
	if !p.enabled() {
		return
	}

	p.logger.Debug("Remove password history")
	err = p.passwordHistoryStore.RemovePasswordHistory(authID, p.pwHistorySize, p.pwHistoryDays)
	if err != nil {
		p.logger.WithError(err).Error("Unable to housekeep password history")
	}

	return
}

func (p *PwHousekeeper) enabled() bool {
	return p.passwordHistoryEnabled
}
