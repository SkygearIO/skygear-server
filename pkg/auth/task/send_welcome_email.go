package task

import (
	"context"
	"net/url"

	"github.com/skygeario/skygear-server/pkg/auth"
	"github.com/skygeario/skygear-server/pkg/auth/dependency/welcemail"
	"github.com/skygeario/skygear-server/pkg/auth/model"
	"github.com/skygeario/skygear-server/pkg/core/async"
	"github.com/skygeario/skygear-server/pkg/core/auth/userprofile"
	"github.com/skygeario/skygear-server/pkg/core/db"
	"github.com/skygeario/skygear-server/pkg/core/errors"
	"github.com/skygeario/skygear-server/pkg/core/inject"

	"github.com/sirupsen/logrus"
)

const (
	// WelcomeEmailSendTaskName provides the name for submiting WelcomeEmailSendTask
	WelcomeEmailSendTaskName = "WelcomeEmailSendTask"
)

func AttachWelcomeEmailSendTask(
	executor *async.Executor,
	authDependency auth.DependencyMap,
) *async.Executor {
	executor.Register(WelcomeEmailSendTaskName, &WelcomeEmailSendTaskFactory{
		authDependency,
	})
	return executor
}

type WelcomeEmailSendTaskFactory struct {
	DependencyMap auth.DependencyMap
}

func (c *WelcomeEmailSendTaskFactory) NewTask(ctx context.Context, taskCtx async.TaskContext) async.Task {
	task := &WelcomeEmailSendTask{}
	inject.DefaultTaskInject(task, c.DependencyMap, ctx, taskCtx)
	return async.TxTaskToTask(task, task.TxContext)
}

type WelcomeEmailSendTask struct {
	WelcomeEmailSender welcemail.Sender  `dependency:"WelcomeEmailSender"`
	UserProfileStore   userprofile.Store `dependency:"UserProfileStore"`
	TxContext          db.TxContext      `dependency:"TxContext"`
	Logger             *logrus.Entry     `dependency:"HandlerLogger"`
}

type WelcomeEmailSendTaskParam struct {
	URLPrefix *url.URL
	Email     string
	User      model.User
}

func (w *WelcomeEmailSendTask) WithTx() bool {
	return true
}

func (w *WelcomeEmailSendTask) Run(param interface{}) (err error) {
	taskParam := param.(WelcomeEmailSendTaskParam)

	w.Logger.WithFields(logrus.Fields{"user_id": taskParam.User.ID}).Debug("Sending welcome email")

	if err = w.WelcomeEmailSender.Send(taskParam.URLPrefix, taskParam.Email, taskParam.User); err != nil {
		err = errors.WithDetails(err, errors.Details{"user_id": taskParam.User.ID})
		return
	}

	return
}
