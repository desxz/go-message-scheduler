package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-playground/validator/v10"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
)

const (
	StatusRunning = "running"
	StatusPaused  = "paused"
)

type PoolConfig struct {
	NumWorkers      int           `mapstructure:"numWorkers"`
	Timeout         time.Duration `mapstructure:"timeout"`
	InitialJobFetch bool          `mapstructure:"initialJobFetch"`
}

type WorkerPoolImpl struct {
	numWorkers int
	logger     *zap.Logger
	wg         *sync.WaitGroup
	poolCtx    context.Context
	poolCancel context.CancelFunc

	canFetchNewJobsMutex sync.Mutex
	canFetchNewJobs      bool

	workerMessageStore WorkerMessageStore
	webhookClient      WebhookClient
	workerMessageCache WorkerMessageCache
	appConfig          Config
	validate           *validator.Validate
}

func NewWorkerPool(
	numWorkers int,
	store WorkerMessageStore,
	whClient WebhookClient,
	cache WorkerMessageCache,
	cfg Config,
	logger *zap.Logger,
	wg *sync.WaitGroup,
	canFetchNewJobsInitial bool,
	validate *validator.Validate,
) *WorkerPoolImpl {
	ctx, cancel := context.WithCancel(context.Background())
	return &WorkerPoolImpl{
		numWorkers:         numWorkers,
		logger:             logger.With(zap.String("component", "workerpool")),
		poolCtx:            ctx,
		poolCancel:         cancel,
		workerMessageStore: store,
		webhookClient:      whClient,
		workerMessageCache: cache,
		appConfig:          cfg,
		canFetchNewJobs:    canFetchNewJobsInitial,
		wg:                 wg,
		validate:           validate,
	}
}

func (p *WorkerPoolImpl) Start() {
	if p.numWorkers <= 0 {
		return
	}

	for i := 0; i < p.numWorkers; i++ {
		p.wg.Add(1)
		workerID := fmt.Sprintf("worker-%d-%s", i+1, primitive.NewObjectID().Hex())

		instance := NewWorkerInstance(
			workerID,
			p.workerMessageStore,
			p.webhookClient,
			p.workerMessageCache,
			p.appConfig.Worker,
			p.logger,
			p.validate,
		)

		canFetchFunc := func() bool {
			p.canFetchNewJobsMutex.Lock()
			defer p.canFetchNewJobsMutex.Unlock()
			return p.canFetchNewJobs
		}

		go instance.Start(context.Background(), p.wg, canFetchFunc)
	}
}

func (p *WorkerPoolImpl) ResumeFetching() {
	p.canFetchNewJobsMutex.Lock()
	defer p.canFetchNewJobsMutex.Unlock()
	if p.canFetchNewJobs {
		p.logger.Info("Worker'lar zaten yeni iş alıyor (aktif durumda).")
		return
	}
	p.logger.Info("Worker'ların yeni iş alması aktif ediliyor...")
	p.canFetchNewJobs = true
}

func (p *WorkerPoolImpl) PauseFetching() {
	p.canFetchNewJobsMutex.Lock()
	defer p.canFetchNewJobsMutex.Unlock()
	if !p.canFetchNewJobs {
		p.logger.Info("Worker'lar zaten yeni iş almıyor (duraklatılmış durumda).")
		return
	}
	p.logger.Info("Worker'ların yeni iş alması duraklatılıyor...")
	p.canFetchNewJobs = false
}

func (p *WorkerPoolImpl) GetStatus() string {
	p.canFetchNewJobsMutex.Lock()
	defer p.canFetchNewJobsMutex.Unlock()
	if p.canFetchNewJobs {
		return StatusRunning
	}
	return StatusPaused
}

func (p *WorkerPoolImpl) Shutdown(timeoutCtx context.Context) error {
	p.PauseFetching()

	p.poolCancel()

	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-timeoutCtx.Done():
		return timeoutCtx.Err()
	}
}
