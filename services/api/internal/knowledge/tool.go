package knowledge

import (
	"context"
	"encoding/json"
	"fmt"

	"nexusai/services/api/internal/agent"
)

const searchKnowledgeToolName = "search_knowledge"

var searchKnowledgeParameters = json.RawMessage(`{
  "type": "object",
  "properties": {
    "query": {
      "type": "string",
      "minLength": 1,
      "maxLength": 1000,
      "description": "需要检索的自然语言问题或关键词"
    },
    "limit": {
      "type": "integer",
      "minimum": 1,
      "maximum": 10,
      "description": "返回的最相关知识分块数量，默认 5"
    }
  },
  "required": ["query"],
  "additionalProperties": false
}`)

type searchKnowledgeArguments struct {
	Query string `json:"query"`
	Limit int    `json:"limit,omitempty"`
}

type searchKnowledgeResult struct {
	Matches []searchKnowledgeMatch `json:"matches"`
}

type searchKnowledgeMatch struct {
	DocumentID string  `json:"documentId"`
	ChunkID    string  `json:"chunkId"`
	Ordinal    int     `json:"ordinal"`
	Content    string  `json:"content"`
	Distance   float64 `json:"distance"`
}

func (s *Service) Tool() agent.Tool {
	return agent.Tool{
		Name:        searchKnowledgeToolName,
		Description: "检索企业知识库中最相关的文档分块。回答知识问题时使用；只能依据返回内容作答并标注 documentId，没有匹配结果时明确说明未找到。",
		Parameters:  searchKnowledgeParameters,
		Execute:     s.executeSearch,
	}
}

func (s *Service) executeSearch(ctx context.Context, proposal agent.ToolProposal) (string, error) {
	var arguments searchKnowledgeArguments
	if err := json.Unmarshal(proposal.Arguments, &arguments); err != nil {
		return "", fmt.Errorf("invalid search_knowledge arguments: %w", err)
	}
	matches, err := s.Search(ctx, arguments.Query, arguments.Limit)
	if err != nil {
		return "", err
	}
	result := searchKnowledgeResult{Matches: make([]searchKnowledgeMatch, 0, len(matches))}
	for _, match := range matches {
		result.Matches = append(result.Matches, searchKnowledgeMatch{
			DocumentID: match.DocumentID,
			ChunkID:    match.ChunkID,
			Ordinal:    match.Ordinal,
			Content:    match.Content,
			Distance:   match.Distance,
		})
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("encode knowledge search result: %w", err)
	}
	return string(encoded), nil
}
