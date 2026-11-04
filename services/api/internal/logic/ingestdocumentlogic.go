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

type IngestDocumentLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewIngestDocumentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *IngestDocumentLogic {
	return &IngestDocumentLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

var ErrKnowledgeNotConfigured = errors.New("knowledge ingestion is not configured")

func (l *IngestDocumentLogic) IngestDocument(req *types.IngestDocumentInput) (*types.IngestDocumentResponse, error) {
	if l.svcCtx.Knowledge == nil {
		return nil, ErrKnowledgeNotConfigured
	}
	documentID, err := l.svcCtx.Knowledge.IngestDocument(l.ctx, req.Title, req.Content, auth.TenantIDFromContext(l.ctx))
	if err != nil {
		return nil, err
	}
	return &types.IngestDocumentResponse{DocumentID: documentID}, nil
}
