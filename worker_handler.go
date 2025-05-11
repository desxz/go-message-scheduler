package main

import (
	"github.com/gofiber/fiber/v2"
)

type WorkerPool interface {
	ResumeFetching()
	PauseFetching()
	GetStatus() string
}

type WorkerPoolHandler struct {
	workerPool WorkerPool
}

type WorkerPoolStatusResponse struct {
	Status string `json:"status"`
}

type WorkerPoolActionRequest struct {
	Action string `json:"action"` // "start" or "pause"
}

func NewWorkerPoolHandler(wp WorkerPool) *WorkerPoolHandler {
	return &WorkerPoolHandler{
		workerPool: wp,
	}
}

func (h *WorkerPoolHandler) RegisterRoutes(app *fiber.App) {
	workerGroup := app.Group("/worker-pool")
	workerGroup.Put("/state", h.ControlWorkerPool)
}

// ControlWorkerPool godoc
// @Summary Updates the worker pool state
// @Description Start or pause the worker pool
// @Tags worker-pool
// @Accept json
// @Produce json
// @Param action body WorkerPoolActionRequest true "Action to perform `start` or `pause`"
// @Success 200 {object} WorkerPoolStatusResponse
// @Failure 400 {object} map[string]string "Invalid action"
// @Router /worker-pool/state [put]
func (h *WorkerPoolHandler) ControlWorkerPool(c *fiber.Ctx) error {
	var req WorkerPoolActionRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	switch req.Action {
	case "start":
		h.workerPool.ResumeFetching()
	case "pause":
		h.workerPool.PauseFetching()
	default:
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid action. Use 'start' or 'pause'",
		})
	}

	return c.JSON(WorkerPoolStatusResponse{
		Status: h.workerPool.GetStatus(),
	})
}
