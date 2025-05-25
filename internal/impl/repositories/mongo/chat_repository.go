package repositories_mongo

import (
	"context"
	"time"

	"aiagent/internal/domain/entities"
	"aiagent/internal/domain/errs"
	"aiagent/internal/domain/interfaces"

	"go.mongodb.org/mongo-driver/bson"
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

func (r *MongoChatRepository) ListChats(ctx context.Context) ([]*entities.Chat, error) {
	var chats []*entities.Chat
	cursor, err := r.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, errors.InternalErrorf("failed to list chats: %v", err)
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var chat entities.Chat
		if err := cursor.Decode(&chat); err != nil {
			return nil, errors.InternalErrorf("failed to decode chat: %v", err)
		}
		chats = append(chats, &chat)
	}

	if err := cursor.Err(); err != nil {
		return nil, errors.InternalErrorf("failed to list chats: %v", err)
	}

	return chats, nil
}

func (r *MongoChatRepository) GetChat(ctx context.Context, id string) (*entities.Chat, error) {
	var chat entities.Chat
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&chat)
	if err == mongo.ErrNoDocuments {
		return nil, errors.NotFoundErrorf("chat not found: %s", id)
	}
	if err != nil {
		return nil, err
	}

	return &chat, nil
}

func (r *MongoChatRepository) CreateChat(ctx context.Context, chat *entities.Chat) error {
	_, err := r.collection.InsertOne(ctx, chat)
	if err != nil {
		return errors.InternalErrorf("failed to create chat: %v", err)
	}

	return nil
}

func (r *MongoChatRepository) UpdateChat(ctx context.Context, chat *entities.Chat) error {
	chat.UpdatedAt = time.Now()

	update, err := bson.Marshal(bson.M{
		"$set": chat,
	})
	if err != nil {
		return errors.InternalErrorf("failed to marshal chat: %v", err)
	}

	result, err := r.collection.UpdateOne(ctx, bson.M{"_id": chat.ID}, update)
	if err != nil {
		return errors.InternalErrorf("failed to update chat: %v", err)
	}
	if result.MatchedCount == 0 {
		return errors.NotFoundErrorf("chat not found: %s", chat.ID)
	}

	return nil
}

func (r *MongoChatRepository) DeleteChat(ctx context.Context, id string) error {
	result, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return errors.InternalErrorf("failed to delete chat: %v", err)
	}
	if result.DeletedCount == 0 {
		return errors.NotFoundErrorf("chat not found: %s", id)
	}

	return nil
}

var _ interfaces.ChatRepository = (*MongoChatRepository)(nil)
