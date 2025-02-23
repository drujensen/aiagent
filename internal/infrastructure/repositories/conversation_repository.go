package repositories

import (
	"context"
	"time"

	"aiagent/internal/domain/entities"
	"aiagent/internal/domain/interfaces"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type MongoConversationRepository struct {
	collection *mongo.Collection
}

func NewMongoConversationRepository(collection *mongo.Collection) *MongoConversationRepository {
	return &MongoConversationRepository{
		collection: collection,
	}
}

func (r *MongoConversationRepository) CreateConversation(ctx context.Context, conversation *entities.Conversation) error {
	conversation.CreatedAt = time.Now()
	conversation.UpdatedAt = time.Now()
	conversation.ID = primitive.NewObjectID()

	result, err := r.collection.InsertOne(ctx, conversation)
	if err != nil {
		return err
	}

	if oid, ok := result.InsertedID.(primitive.ObjectID); ok {
		conversation.ID = oid
	} else {
		return mongo.ErrInvalidIndexValue
	}

	return nil
}

func (r *MongoConversationRepository) GetConversation(ctx context.Context, id string) (*entities.Conversation, error) {
	var conversation entities.Conversation
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, mongo.ErrNoDocuments
	}

	err = r.collection.FindOne(ctx, bson.M{"_id": oid}).Decode(&conversation)
	if err == mongo.ErrNoDocuments {
		return nil, mongo.ErrNoDocuments
	}
	if err != nil {
		return nil, err
	}

	return &conversation, nil
}

func (r *MongoConversationRepository) UpdateConversation(ctx context.Context, conversation *entities.Conversation) error {
	conversation.UpdatedAt = time.Now()

	oid, err := primitive.ObjectIDFromHex(conversation.ID.Hex())
	if err != nil {
		return mongo.ErrNoDocuments
	}

	update := bson.M{
		"$set": bson.M{
			"agent_id":   conversation.AgentID,
			"messages":   conversation.Messages,
			"updated_at": conversation.UpdatedAt,
			"active":     conversation.Active,
		},
	}

	result, err := r.collection.UpdateOne(ctx, bson.M{"_id": oid}, update)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}

	return nil
}

func (r *MongoConversationRepository) DeleteConversation(ctx context.Context, id string) error {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return mongo.ErrNoDocuments
	}

	result, err := r.collection.DeleteOne(ctx, bson.M{"_id": oid})
	if err != nil {
		return err
	}
	if result.DeletedCount == 0 {
		return mongo.ErrNoDocuments
	}

	return nil
}

func (r *MongoConversationRepository) ListConversations(ctx context.Context) ([]*entities.Conversation, error) {
	var conversations []*entities.Conversation
	cursor, err := r.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var conversation entities.Conversation
		if err := cursor.Decode(&conversation); err != nil {
			return nil, err
		}
		conversations = append(conversations, &conversation)
	}

	if err := cursor.Err(); err != nil {
		return nil, err
	}

	return conversations, nil
}

var _ interfaces.ConversationRepository = (*MongoConversationRepository)(nil)
