package async

import (
	"context"
	"fmt"

	"github.com/skygeario/skygear-server/pkg/core/db"
	"github.com/skygeario/skygear-server/pkg/core/logging"
)

type Executor struct {
	taskFactoryMap map[string]TaskFactory
	pool           db.Pool
}

func NewExecutor(dbPool db.Pool) *Executor {
	return &Executor{
		taskFactoryMap: map[string]TaskFactory{},
		pool:           dbPool,
	}
}

func (e *Executor) Register(name string, taskFactory TaskFactory) {
	e.taskFactoryMap[name] = taskFactory
}

func (e *Executor) Execute(taskCtx TaskContext, name string, param interface{}, response chan error) {
	factory := e.taskFactoryMap[name]
	ctx := db.InitDBContext(context.Background(), e.pool)
	task := factory.NewTask(ctx, taskCtx)

	formatter := logging.NewDefaultMaskedTextFormatter(taskCtx.TenantConfig.DefaultSensitiveLoggerValues())
	loggerFactory := logging.NewFactoryFromRequestID(taskCtx.RequestID, formatter)
	logger := loggerFactory.NewLogger("async-executor")
	go func() {
		defer func() {
			if rec := recover(); rec != nil {
				logger.WithFields(map[string]interface{}{
					"task_name": name,
					"error":     rec,
				}).Error("unexpected error occurred when running async task")

				if response != nil {
					switch err := rec.(type) {
					case error:
						response <- err
					default:
						response <- fmt.Errorf("%+v", err)
					}
				}
			}
		}()

		err := task.Run(param)
		if response != nil {
			response <- err
		}
	}()
}