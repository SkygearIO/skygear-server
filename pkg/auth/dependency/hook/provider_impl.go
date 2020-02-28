package hook

import (
	"fmt"
	"net/url"
	gotime "time"

	"github.com/sirupsen/logrus"

	"github.com/skygeario/skygear-server/pkg/core/auth"
	"github.com/skygeario/skygear-server/pkg/core/auth/authinfo"
	"github.com/skygeario/skygear-server/pkg/core/auth/event"
	"github.com/skygeario/skygear-server/pkg/core/auth/model"
	"github.com/skygeario/skygear-server/pkg/core/auth/model/format"
	"github.com/skygeario/skygear-server/pkg/core/auth/userprofile"
	"github.com/skygeario/skygear-server/pkg/core/errors"
	"github.com/skygeario/skygear-server/pkg/core/logging"
	"github.com/skygeario/skygear-server/pkg/core/skyerr"
	"github.com/skygeario/skygear-server/pkg/core/time"
	"github.com/skygeario/skygear-server/pkg/core/urlprefix"
)

type providerImpl struct {
	RequestID               string
	BaseURL                 *url.URL
	Store                   Store
	AuthContext             auth.ContextGetter
	TimeProvider            time.Provider
	AuthInfoStore           authinfo.Store
	UserProfileStore        userprofile.Store
	Deliverer               Deliverer
	PersistentEventPayloads []event.Payload
	Logger                  *logrus.Entry
}

func NewProvider(
	requestID string,
	urlprefix urlprefix.Provider,
	store Store,
	authContext auth.ContextGetter,
	timeProvider time.Provider,
	authInfoStore authinfo.Store,
	userProfileStore userprofile.Store,
	deliverer Deliverer,
	loggerFactory logging.Factory,
) Provider {
	return &providerImpl{
		RequestID:        requestID,
		BaseURL:          urlprefix.Value(),
		Store:            store,
		AuthContext:      authContext,
		TimeProvider:     timeProvider,
		AuthInfoStore:    authInfoStore,
		UserProfileStore: userProfileStore,
		Deliverer:        deliverer,
		Logger:           loggerFactory.NewLogger("hook"),
	}
}

func (provider *providerImpl) DispatchEvent(payload event.Payload, user *model.User) (err error) {
	var seq int64
	switch typedPayload := payload.(type) {
	case event.OperationPayload:
		if provider.Deliverer.WillDeliver(typedPayload.BeforeEventType()) {
			seq, err = provider.Store.NextSequenceNumber()
			if err != nil {
				err = errors.HandledWithMessage(err, "failed to dispatch event")
				return
			}
			event := event.NewBeforeEvent(seq, typedPayload, provider.makeContext())
			err = provider.Deliverer.DeliverBeforeEvent(provider.BaseURL, event, user)
			if err != nil {
				if !skyerr.IsKind(err, WebHookDisallowed) {
					err = errors.HandledWithMessage(err, "failed to dispatch event")
				}
				return
			}

			// update payload since it may have been updated by mutations
			payload = event.Payload
		}

		provider.PersistentEventPayloads = append(provider.PersistentEventPayloads, payload)
		return

	case event.NotificationPayload:
		provider.PersistentEventPayloads = append(provider.PersistentEventPayloads, payload)
		err = nil
		return

	default:
		panic(fmt.Sprintf("hook: invalid event payload: %T", payload))
	}
}

func (provider *providerImpl) WillCommitTx() error {
	err := provider.dispatchSyncUserEventIfNeeded()
	if err != nil {
		return err
	}

	events := []*event.Event{}
	for _, payload := range provider.PersistentEventPayloads {
		var ev *event.Event

		switch typedPayload := payload.(type) {
		case event.OperationPayload:
			if provider.Deliverer.WillDeliver(typedPayload.AfterEventType()) {
				seq, err := provider.Store.NextSequenceNumber()
				if err != nil {
					err = errors.HandledWithMessage(err, "failed to persist event")
					return err
				}
				ev = event.NewAfterEvent(seq, typedPayload, provider.makeContext())
			}

		case event.NotificationPayload:
			if provider.Deliverer.WillDeliver(typedPayload.EventType()) {
				seq, err := provider.Store.NextSequenceNumber()
				if err != nil {
					err = errors.HandledWithMessage(err, "failed to persist event")
					return err
				}
				ev = event.NewEvent(seq, typedPayload, provider.makeContext())
			}

		default:
			panic(fmt.Sprintf("hook: invalid event payload: %T", payload))
		}

		if ev == nil {
			continue
		}
		events = append(events, ev)
	}

	err = provider.Store.AddEvents(events)
	if err != nil {
		err = errors.HandledWithMessage(err, "failed to persist event")
		return err
	}
	provider.PersistentEventPayloads = nil

	return nil
}

func (provider *providerImpl) DidCommitTx() {
	// TODO(webhook): deliver persisted events
	events, _ := provider.Store.GetEventsForDelivery()
	for _, event := range events {
		err := provider.Deliverer.DeliverNonBeforeEvent(provider.BaseURL, event, 60*gotime.Second)
		if err != nil {
			provider.Logger.WithError(err).Debug("Failed to dispatch event")
		}
	}
}

func (provider *providerImpl) dispatchSyncUserEventIfNeeded() error {
	userIDToSync := []string{}

	for _, payload := range provider.PersistentEventPayloads {
		if _, isOperation := payload.(event.OperationPayload); !isOperation {
			continue
		}
		if userAwarePayload, ok := payload.(event.UserAwarePayload); ok {
			userIDToSync = append(userIDToSync, userAwarePayload.UserID())
		}
	}

	for _, userID := range userIDToSync {
		var authInfo authinfo.AuthInfo
		err := provider.AuthInfoStore.GetAuth(userID, &authInfo)
		if err != nil {
			return err
		}

		userProfile, err := provider.UserProfileStore.GetUserProfile(userID)
		if err != nil {
			return err
		}

		user := model.NewUser(authInfo, userProfile)
		payload := event.UserSyncEvent{User: user}
		err = provider.DispatchEvent(payload, &user)
		if err != nil {
			return err
		}
	}

	return nil
}

func (provider *providerImpl) makeContext() event.Context {
	var requestID, userID, principalID *string
	var session *model.Session

	if provider.RequestID == "" {
		requestID = nil
	} else {
		requestID = &provider.RequestID
	}

	authInfo, _ := provider.AuthContext.AuthInfo()
	sess, _ := provider.AuthContext.Session()
	if authInfo == nil {
		userID = nil
		principalID = nil
		session = nil
	} else {
		userID = &authInfo.ID
		principalID = &sess.PrincipalID
		s := format.SessionFromSession(sess)
		session = &s
	}

	return event.Context{
		Timestamp:   provider.TimeProvider.NowUTC().Unix(),
		RequestID:   requestID,
		UserID:      userID,
		PrincipalID: principalID,
		Session:     session,
	}
}
