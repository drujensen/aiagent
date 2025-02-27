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

type MongoChatRepository struct {
	collection *mongo.Collection
}

func NewMongoChatRepository(collection *mongo.Collection) *MongoChatRepository {
	return &MongoChatRepository{
		collection: collection,
	}
}

func (r *MongoChatRepository) CreateChat(ctx context.Context, chat *entities.Chat) error {
	chat.CreatedAt = time.Now()
	chat.UpdatedAt = time.Now()
	chat.ID = primitive.NewObjectID()

	result, err := r.collection.InsertOne(ctx, chat)
	if err != nil {
		return err
	}

	if oid, ok := result.InsertedID.(primitive.ObjectID); ok {
		chat.ID = oid
	} else {
		return mongo.ErrInvalidIndexValue
	}

	return nil
}

func (r *MongoChatRepository) GetChat(ctx context.Context, id string) (*entities.Chat, error) {
	var chat entities.Chat
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, mongo.ErrNoDocuments
	}

	err = r.collection.FindOne(ctx, bson.M{"_id": oid}).Decode(&chat)
	if err == mongo.ErrNoDocuments {
		return nil, mongo.ErrNoDocuments
	}
	if err != nil {
		return nil, err
	}

	return &chat, nil
}

func (r *MongoChatRepository) UpdateChat(ctx context.Context, chat *entities.Chat) error {
	chat.UpdatedAt = time.Now()

	oid, err := primitive.ObjectIDFromHex(chat.ID.Hex())
	if err != nil {
		return mongo.ErrNoDocuments
	}

	update := bson.M{
		"$set": bson.M{
			"agent_id":   chat.AgentID,
			"name":       chat.Name, // Update the name in MongoDB
			"messages":   chat.Messages,
			"updated_at": chat.UpdatedAt,
			"active":     chat.Active,
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

func (r *MongoChatRepository) DeleteChat(ctx context.Context, id string) error {
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

func (r *MongoChatRepository) ListChats(ctx context.Context) ([]*entities.Chat, error) {
	var chats []*entities.Chat
	cursor, err := r.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var chat entities.Chat
		if err := cursor.Decode(&chat); err != nil {
			return nil, err
		}
		chats = append(chats, &chat)
	}

	if err := cursor.Err(); err != nil {
		return nil, err
	}

	return chats, nil
}

var _ interfaces.ChatRepository = (*MongoChatRepository)(nil)
