// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package logic

import (
	"context"
	"errors"

	"nexusai/services/api/internal/auth"
	"nexusai/services/api/internal/svc"
	"nexusai/services/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListKnowledgeDocumentsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListKnowledgeDocumentsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListKnowledgeDocumentsLogic {
	return &ListKnowledgeDocumentsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListKnowledgeDocumentsLogic) ListKnowledgeDocuments() (*types.ListKnowledgeDocumentsResponse, error) {
	if l.svcCtx.Documents == nil {
		return nil, errors.New("knowledge document store is not configured")
	}
	documents, err := l.svcCtx.Documents.ListDocuments(l.ctx, auth.TenantIDFromContext(l.ctx))
	if err != nil {
		return nil, err
	}
	return &types.ListKnowledgeDocumentsResponse{Documents: knowledgeDocumentResponses(documents)}, nil
}
