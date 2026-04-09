package knowledge

import (
	"context"

	"modulr/events"
)

// KnowledgeAPI публичный контракт модуля знаний.
type KnowledgeAPI interface {
	SaveArticle(ctx context.Context, a *Article) error
	Query(ctx context.Context, q string, segment events.Segment) (*QueryResult, error)
	VerifyFact(ctx context.Context, claim string) (*FactCheck, error)
	UpsertTopicGraph(ctx context.Context, g *TopicGraph) error
	RegisterSubscriptions(bus *events.EventBus)
}
