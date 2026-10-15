package gemini

import (
	"slices"
	"sync"

	"google.golang.org/genai"
)

// modelTurnCache keeps Gemini's model turns that ended in tool calls, keyed by tool call ID.
//
// langchaingo messages have no place for thought signatures, so an agent's next request would send the tool calls
// without them, and Gemini 3 rejects that with a 400. The adapter restores the original turn (parts and signatures,
// in order) from here. Entries are per process: a loop that continues on another instance falls back to the turn
// rebuilt from langchaingo's message, with models.GeminiSkipThoughtSignature.
type modelTurnCache struct {
	mu      sync.Mutex
	maxSize int
	turns   map[string]cachedModelTurn
	order   []string
}

type cachedModelTurn struct {
	turn        *genai.Content
	toolCallIDs []string
	// geminiIDs maps our tool call IDs to the IDs Gemini gave its function calls ("" when it gave none)
	geminiIDs map[string]string
}

func newModelTurnCache(maxSize int) *modelTurnCache {
	return &modelTurnCache{maxSize: maxSize, turns: map[string]cachedModelTurn{}}
}

func (c *modelTurnCache) store(entry cachedModelTurn) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, id := range entry.toolCallIDs {
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
func (c *modelTurnCache) find(toolCallIDs []string) (cachedModelTurn, bool) {
	if len(toolCallIDs) == 0 {
		return cachedModelTurn{}, false
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	entry, ok := c.turns[toolCallIDs[0]]
	if !ok || !slices.Equal(entry.toolCallIDs, toolCallIDs) {
		return cachedModelTurn{}, false
	}
	return entry, true
}

// geminiID returns the ID Gemini gave the function call, for its function response
func (c *modelTurnCache) geminiID(toolCallID string) string {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.turns[toolCallID].geminiIDs[toolCallID]
}
