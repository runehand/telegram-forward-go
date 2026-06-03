package mongo

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"zenfl-forwarder/apps/backend/internal/config"
	"zenfl-forwarder/apps/backend/internal/domain"
)

type Store struct {
	client   *mongo.Client
	db       *mongo.Database
	msgs     *mongo.Collection
	users    *mongo.Collection
}

func New(ctx context.Context, cfg config.MongoConfig) (*Store, error) {
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.URI))
	if err != nil {
		return nil, err
	}
	db := client.Database(cfg.Database)
	return &Store{
		client: client,
		db:     db,
		msgs:   db.Collection(cfg.Collection),
		users:  db.Collection(cfg.UsersColl),
	}, nil
}

func (s *Store) Close(ctx context.Context) error { return s.client.Disconnect(ctx) }

func (s *Store) SaveMessage(ctx context.Context, msg domain.JobMessage) error {
	if msg.ReceivedAt.IsZero() {
		msg.ReceivedAt = time.Now().UTC()
	}
	_, err := s.msgs.InsertOne(ctx, msg)
	return err
}

func (s *Store) ListMessages(ctx context.Context, limit int64) ([]domain.JobMessage, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	opts := options.Find().SetLimit(limit).SetSort(bson.D{{Key: "received_at", Value: -1}})
	cur, err := s.msgs.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var out []domain.JobMessage
	if err := cur.All(ctx, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Store) EnsureDemoUser(ctx context.Context, email, passwordHash string) error {
	_, err := s.users.UpdateOne(ctx, bson.M{"email": email}, bson.M{"$setOnInsert": bson.M{"email": email, "password_hash": passwordHash}}, options.Update().SetUpsert(true))
	return err
}

func (s *Store) FindUserByEmail(ctx context.Context, email string) (domain.User, error) {
	var u domain.User
	err := s.users.FindOne(ctx, bson.M{"email": email}).Decode(&u)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return domain.User{}, errors.New("invalid credentials")
		}
		return domain.User{}, err
	}
	return u, nil
}
