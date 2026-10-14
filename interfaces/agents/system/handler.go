package system_agent

import (
	"context"

	"go-serviceboilerplate/applications/usecases"
	"go-serviceboilerplate/infrastructures/langchain/agents"
)

// SystemAgentTools groups the agent tools of the system domain. Like an HTTP handler, each tool decodes and
// validates its input, then calls a use case; it holds no business logic itself
type SystemAgentTools struct {
	systemUsecase *usecases.SystemUsecase
}

func NewSystemAgentTools(systemUsecase *usecases.SystemUsecase) *SystemAgentTools {
	return &SystemAgentTools{systemUsecase}
}

// RegisterTools returns the tools this domain gives to agents. Mirrors RegisterRoutes in interfaces/http
func (h *SystemAgentTools) RegisterTools() []agents.Tool {
	return []agents.Tool{
		agents.NewTool("get_system_info",
			"Returns the service name, version and status. Use it when asked about the service itself.",
			h.GetSystemInfo,
		),
	}
}

// GetSystemInfo takes no input (struct{}). Tools with input use a models.<Name>ToolInput struct with json,
// validate and description tags, the same way HTTP handlers use request models
func (h *SystemAgentTools) GetSystemInfo(ctx context.Context, _ struct{}) (string, error) {
	return agents.JSON(h.systemUsecase.GetSystemInfo(ctx))
}
