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

type GetKnowledgeDocumentLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetKnowledgeDocumentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetKnowledgeDocumentLogic {
	return &GetKnowledgeDocumentLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetKnowledgeDocumentLogic) GetKnowledgeDocument(req *types.GetKnowledgeDocumentInput) (*types.KnowledgeDocumentDetail, error) {
	if l.svcCtx.Documents == nil {
		return nil, errors.New("knowledge document store is not configured")
	}
	document, err := l.svcCtx.Documents.GetDocument(l.ctx, req.ID)
	if err != nil {
		return nil, err
	}
	response := knowledgeDocumentDetailResponse(document)
	return &response, nil
}
