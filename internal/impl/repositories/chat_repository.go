package repositories

import (
	"context"
	"time"

	"aiagent/internal/domain/entities"
	"aiagent/internal/domain/errors"
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
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, errors.ValidationErrorf("invalid chat ID: %s", id)
	}

	err = r.collection.FindOne(ctx, bson.M{"_id": oid}).Decode(&chat)
	if err == mongo.ErrNoDocuments {
		return nil, errors.NotFoundErrorf("chat not found: %s", id)
	}
	if err != nil {
		return nil, err
	}

	return &chat, nil
}

func (r *MongoChatRepository) CreateChat(ctx context.Context, chat *entities.Chat) error {
	chat.CreatedAt = time.Now()
	chat.UpdatedAt = time.Now()
	chat.ID = primitive.NewObjectID()

	result, err := r.collection.InsertOne(ctx, chat)
	if err != nil {
		return errors.InternalErrorf("failed to create chat: %v", err)
	}

	if oid, ok := result.InsertedID.(primitive.ObjectID); ok {
		chat.ID = oid
	} else {
		return errors.ValidationErrorf("failed to convert InsertedID to ObjectID")
	}

	return nil
}

func (r *MongoChatRepository) UpdateChat(ctx context.Context, chat *entities.Chat) error {
	chat.UpdatedAt = time.Now()

	oid, err := primitive.ObjectIDFromHex(chat.ID.Hex())
	if err != nil {
		return errors.ValidationErrorf("invalid chat ID: %s", chat.ID.Hex())
	}

	// Convert the chat struct to BSON
	update, err := bson.Marshal(bson.M{
		"$set": chat,
	})
	if err != nil {
		return errors.InternalErrorf("failed to marshal chat: %v", err)
	}

	result, err := r.collection.UpdateOne(ctx, bson.M{"_id": oid}, update)
	if err != nil {
		return errors.InternalErrorf("failed to update chat: %v", err)
	}
	if result.MatchedCount == 0 {
		return errors.NotFoundErrorf("chat not found: %s", chat.ID.Hex())
	}

	return nil
}

func (r *MongoChatRepository) DeleteChat(ctx context.Context, id string) error {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.ValidationErrorf("invalid chat ID: %s", id)
	}

	result, err := r.collection.DeleteOne(ctx, bson.M{"_id": oid})
	if err != nil {
		return errors.InternalErrorf("failed to delete chat: %v", err)
	}
	if result.DeletedCount == 0 {
		return errors.NotFoundErrorf("chat not found: %s", id)
	}

	return nil
}

var _ interfaces.ChatRepository = (*MongoChatRepository)(nil)
