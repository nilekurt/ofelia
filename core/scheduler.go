package core

import (
	"errors"
	"fmt"
	"sync"

	"github.com/robfig/cron/v3"
)

var (
	ErrAlreadyStarted = errors.New("scheduler has already started")
	ErrAlreadyStopped = errors.New("scheduler has already stopped")
	ErrEmptyScheduler = errors.New("unable to start a empty scheduler")
	ErrEmptySchedule  = errors.New("unable to add a job with a empty schedule")
)

type Scheduler struct {
	Jobs   []Job
	Logger Logger

	middlewareContainer
	cron      *cron.Cron
	wg        sync.WaitGroup
	isRunning bool
}

func NewScheduler(l Logger) *Scheduler {
	return &Scheduler{
		Logger: l,
		cron:   cron.New(),
	}
}

func (s *Scheduler) AddJob(j Job) error {
	s.Logger.Noticef("New job registered %q - %q - %q", j.GetName(), j.GetCommand(), j.GetSchedule())

	if j.GetSchedule() == "" {
		return ErrEmptySchedule
	}

	if _, err := s.cron.AddJob(j.GetSchedule(), &jobWrapper{s, j}); err != nil {
		return err
	}

	s.Jobs = append(s.Jobs, j)
	return nil
}

func (s *Scheduler) Start() error {
	if s.isRunning {
		return ErrAlreadyStarted
	}

	if len(s.Jobs) == 0 {
		return ErrEmptyScheduler
	}

	s.Logger.Debugf("Starting scheduler with %d jobs", len(s.Jobs))

	s.mergeMiddlewares()
	s.isRunning = true
	s.cron.Start()
	return nil
}

func (s *Scheduler) mergeMiddlewares() {
	for _, j := range s.Jobs {
		j.Use(s.Middlewares()...)
	}
}

func (s *Scheduler) Stop() error {
	if !s.isRunning {
		return ErrAlreadyStopped
	}

	s.wg.Wait()
	s.cron.Stop()
	s.isRunning = false
	return nil
}

func (s *Scheduler) IsRunning() bool {
	return s.isRunning
}

type jobWrapper struct {
	s *Scheduler
	j Job
}

func (w *jobWrapper) Run() {
	w.s.wg.Add(1)
	defer w.s.wg.Done()

	e := NewExecution()
	ctx := NewContext(w.s, w.j, e)

	w.start(ctx)
	err := ctx.Next()
	w.stop(ctx, err)
}

func (w *jobWrapper) start(ctx *Context) {
	ctx.Start()
	ctx.Log("Started - " + ctx.Job.GetCommand())
}

func (w *jobWrapper) stop(ctx *Context, err error) {
	ctx.Stop(err)

	errText := "none"
	if ctx.Execution.Error != nil {
		errText = ctx.Execution.Error.Error()
	}

	if ctx.Execution.OutputStream.TotalWritten() > 0 {
		ctx.Log("StdOut: " + ctx.Execution.OutputStream.String())
	}

	if ctx.Execution.ErrorStream.TotalWritten() > 0 {
		ctx.Log("StdErr: " + ctx.Execution.ErrorStream.String())
	}

	msg := fmt.Sprintf(
		"Finished in %q, failed: %t, skipped: %t, error: %s",
		ctx.Execution.Duration, ctx.Execution.Failed, ctx.Execution.Skipped, errText,
	)

	ctx.Log(msg)
}
