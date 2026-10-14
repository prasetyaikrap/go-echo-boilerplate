package agent_tools

import (
	"go-serviceboilerplate/applications/usecases"
	"go-serviceboilerplate/infrastructures/langchain/agents"
	system_agent "go-serviceboilerplate/interfaces/agents/system"
)

// AgentTools gathers the agent tools of every domain, with their use cases injected once at startup in main.go —
// the agent counterpart of registering HTTP routes. Give each agent only the tools its job needs, e.g.
//
//	agentTools := agent_tools.NewAgentTools(systemUsecase)
//	supportAgent := &agents.Agent{Name: "support", LLM: mainLLM, Tools: agentTools.System.RegisterTools()}
type AgentTools struct {
	System *system_agent.SystemAgentTools
}

// NewAgentTools takes the use cases the tools call. Add a parameter and a field per new domain
func NewAgentTools(systemUsecase *usecases.SystemUsecase) *AgentTools {
	return &AgentTools{
		System: system_agent.NewSystemAgentTools(systemUsecase),
	}
}

// All returns every domain's tools, for an agent that may use any of them
func (t *AgentTools) All() []agents.Tool {
	var all []agents.Tool
	all = append(all, t.System.RegisterTools()...)
	return all
}
