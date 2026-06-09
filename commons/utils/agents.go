package utils

import (
	"fmt"
	"go-serviceboilerplate/commons/models"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func GetAgentPromptFileContent(fileName string) (string, error) {
	promptPath := filepath.Join(models.AgentsPromptDirectory, fileName)
	content, err := os.ReadFile(promptPath)
	if err != nil {
		return "", err
	}

	trimmed := strings.TrimSpace(string(content))
	if trimmed == "" {
		return "", fmt.Errorf("prompt file %s is empty", promptPath)
	}

	return trimmed, nil
}

func ApplyPromptPatternReplacements(template string, replacements map[string]string) string {
	if len(replacements) == 0 {
		return template
	}

	keys := make([]string, 0, len(replacements))
	for key := range replacements {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	pairs := make([]string, 0, len(keys)*2)
	for _, key := range keys {
		pairs = append(pairs, "{{"+key+"}}", replacements[key])
	}

	replacer := strings.NewReplacer(pairs...)
	return replacer.Replace(template)
}