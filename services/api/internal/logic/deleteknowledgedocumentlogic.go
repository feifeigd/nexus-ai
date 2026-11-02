// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package logic

import (
	"context"
	"errors"

	"nexusai/services/api/internal/svc"
	"nexusai/services/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteKnowledgeDocumentLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeleteKnowledgeDocumentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteKnowledgeDocumentLogic {
	return &DeleteKnowledgeDocumentLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DeleteKnowledgeDocumentLogic) DeleteKnowledgeDocument(req *types.DeleteKnowledgeDocumentInput) (*types.DeleteKnowledgeDocumentResponse, error) {
	if l.svcCtx.Documents == nil {
		return nil, errors.New("knowledge document store is not configured")
	}
	if err := l.svcCtx.Documents.DeleteDocument(l.ctx, req.ID); err != nil {
		return nil, err
	}
	return &types.DeleteKnowledgeDocumentResponse{DocumentID: req.ID}, nil
}
