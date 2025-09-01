package repositories_mongo

import (
	"context"
	"time"

	"github.com/drujensen/aiagent/internal/domain/entities"
	errors "github.com/drujensen/aiagent/internal/domain/errs"
	"github.com/drujensen/aiagent/internal/domain/interfaces"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type agentSessionRepository struct {
	collection *mongo.Collection
}

func NewAgentSessionRepository(collection *mongo.Collection) interfaces.AgentSessionRepository {
	return &agentSessionRepository{
		collection: collection,
	}
}

func (r *agentSessionRepository) CreateSession(ctx context.Context, session *entities.AgentSession) error {
	_, err := r.collection.InsertOne(ctx, session)
	if err != nil {
		return errors.InternalErrorf("failed to create agent session: %v", err)
	}
	return nil
}

func (r *agentSessionRepository) UpdateSession(ctx context.Context, session *entities.AgentSession) error {
	filter := bson.M{"_id": session.ID}
	update := bson.M{"$set": session}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return errors.InternalErrorf("failed to update agent session: %v", err)
	}
	if result.MatchedCount == 0 {
		return errors.NotFoundErrorf("agent session not found: %s", session.ID)
	}
	return nil
}

func (r *agentSessionRepository) GetSession(ctx context.Context, sessionID string) (*entities.AgentSession, error) {
	var session entities.AgentSession
	err := r.collection.FindOne(ctx, bson.M{"_id": sessionID}).Decode(&session)
	if err == mongo.ErrNoDocuments {
		return nil, errors.NotFoundErrorf("agent session not found: %s", sessionID)
	}
	if err != nil {
		return nil, errors.InternalErrorf("failed to get agent session: %v", err)
	}
	return &session, nil
}

func (r *agentSessionRepository) ListActiveSessions(ctx context.Context, agentID string) ([]*entities.AgentSession, error) {
	filter := bson.M{
		"$or": []bson.M{
			{"parent_agent": agentID},
			{"subagent": agentID},
		},
		"status": bson.M{"$in": []string{"pending", "active"}},
	}

	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, errors.InternalErrorf("failed to list active sessions: %v", err)
	}
	defer cursor.Close(ctx)

	var sessions []*entities.AgentSession
	for cursor.Next(ctx) {
		var session entities.AgentSession
		if err := cursor.Decode(&session); err != nil {
			return nil, errors.InternalErrorf("failed to decode session: %v", err)
		}
		sessions = append(sessions, &session)
	}

	if err := cursor.Err(); err != nil {
		return nil, errors.InternalErrorf("cursor error: %v", err)
	}

	return sessions, nil
}

func (r *agentSessionRepository) CleanupExpiredSessions(ctx context.Context, cutoff time.Time) error {
	filter := bson.M{
		"completed_at": bson.M{"$lt": cutoff},
	}

	_, err := r.collection.DeleteMany(ctx, filter)
	if err != nil {
		return errors.InternalErrorf("failed to cleanup expired sessions: %v", err)
	}

	return nil
}
