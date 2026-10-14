package anthropic

import (
	"slices"
	"sync"

	"github.com/anthropics/anthropic-sdk-go"
)

// assistantTurnCache keeps Claude's assistant turns that ended in tool calls, keyed by tool call ID.
//
// langchaingo messages have no way to carry thinking blocks, so an agent's next request would send the assistant turn
// without them. Claude needs that turn back unchanged (thinking blocks, text and tool calls, in order) to continue a
// tool-use loop, so the adapter restores the original turn from here. Entries are per process: a loop that continues
// on another instance falls back to the turn rebuilt from langchaingo's message.
type assistantTurnCache struct {
	mu      sync.Mutex
	maxSize int
	turns   map[string]cachedTurn
	order   []string
}

type cachedTurn struct {
	turn        anthropic.BetaMessageParam
	toolCallIDs []string
}

func newAssistantTurnCache(maxSize int) *assistantTurnCache {
	return &assistantTurnCache{maxSize: maxSize, turns: map[string]cachedTurn{}}
}

func (c *assistantTurnCache) store(toolCallIDs []string, turn anthropic.BetaMessageParam) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry := cachedTurn{turn: turn, toolCallIDs: slices.Clone(toolCallIDs)}
	for _, id := range toolCallIDs {
		if _, exists := c.turns[id]; !exists {
			c.order = append(c.order, id)
		}
		c.turns[id] = entry
	}

	// Drop the oldest entries once over the limit
	for len(c.order) > c.maxSize {
		delete(c.turns, c.order[0])
		c.order = c.order[1:]
	}
}

// find returns the original turn when it has exactly these tool calls, in this order
func (c *assistantTurnCache) find(toolCallIDs []string) (anthropic.BetaMessageParam, bool) {
	if len(toolCallIDs) == 0 {
		return anthropic.BetaMessageParam{}, false
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	entry, ok := c.turns[toolCallIDs[0]]
	if !ok || !slices.Equal(entry.toolCallIDs, toolCallIDs) {
		return anthropic.BetaMessageParam{}, false
	}

	return entry.turn, true
}
