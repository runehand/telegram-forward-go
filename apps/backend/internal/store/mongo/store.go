package mongo

import (
	"context"
	"errors"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/google/uuid"

	"zenfl-forwarder/apps/backend/internal/config"
	"zenfl-forwarder/apps/backend/internal/domain"
)

type Store struct {
	client *mongo.Client
	msgs   *mongo.Collection
	users  *mongo.Collection
	seen   *mongo.Collection
}

type UserUpsertInput struct {
	Name         string
	Email        string
	Role         domain.UserRole
	PasswordHash string
	Preferences  domain.UserPreferences
}

func New(ctx context.Context, cfg config.MongoConfig) (*Store, error) {
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.URI))
	if err != nil {
		return nil, err
	}
	db := client.Database(cfg.Database)
	s := &Store{
		client: client,
		msgs:   db.Collection(cfg.Collection),
		users:  db.Collection(cfg.UsersColl),
		seen:   db.Collection(cfg.SeenColl),
	}
	if err := s.ensureIndexes(ctx); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) Close(ctx context.Context) error { return s.client.Disconnect(ctx) }

func (s *Store) ensureIndexes(ctx context.Context) error {
	_, err := s.msgs.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "received_at", Value: -1}}},
		{Keys: bson.D{{Key: "title", Value: 1}}},
		{Keys: bson.D{{Key: "skills", Value: 1}}},
		{Keys: bson.D{{Key: "tags", Value: 1}}},
		{Keys: bson.D{{Key: "countries", Value: 1}}},
		{Keys: bson.D{{Key: "only_us", Value: 1}}},
		{Keys: bson.D{{Key: "only_mobile", Value: 1}}},
		{Keys: bson.D{{Key: "payment_verified", Value: 1}}},
	})
	if err != nil {
		return err
	}
	_, err = s.users.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "email", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return err
	}
	_, err = s.seen.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "user_id", Value: 1}, {Key: "job_id", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	return err
}

func (s *Store) SaveJob(ctx context.Context, job domain.Job) error {
	if job.ID == "" {
		job.ID = uuid.NewString()
	}
	if job.ReceivedAt.IsZero() {
		job.ReceivedAt = time.Now().UTC()
	}
	_, err := s.msgs.UpdateOne(
		ctx,
		bson.M{"telegram_msg_id": job.TelegramMsgID, "peer_id": job.PeerID},
		bson.M{"$set": job, "$setOnInsert": bson.M{"_id": job.ID}},
		options.Update().SetUpsert(true),
	)
	return err
}

func (s *Store) ListJobsForUser(ctx context.Context, user domain.User, q domain.JobQuery) ([]domain.Job, error) {
	if q.Limit <= 0 || q.Limit > 200 {
		q.Limit = 50
	}
	filter := bson.M{}
	if q.Query != "" {
		filter["$or"] = []bson.M{
			{"title": bson.M{"$regex": q.Query, "$options": "i"}},
			{"raw_text": bson.M{"$regex": q.Query, "$options": "i"}},
			{"skills": bson.M{"$regex": q.Query, "$options": "i"}},
		}
	}
	if q.OnlyUS {
		filter["only_us"] = true
	}
	if q.OnlyMobile {
		filter["only_mobile"] = true
	}
	if q.Verified != nil {
		filter["payment_verified"] = *q.Verified
	}
	if q.Country != "" {
		filter["countries"] = bson.M{"$regex": "^" + regexpEscape(q.Country) + "$", "$options": "i"}
	}
	if q.Tag != "" {
		filter["tags"] = bson.M{"$regex": "^" + regexpEscape(q.Tag) + "$", "$options": "i"}
	}
	if q.Hours > 0 {
		filter["received_at"] = bson.M{"$gte": time.Now().UTC().Add(-time.Duration(q.Hours) * time.Hour)}
	}
	if q.OnlyUnseen {
		seenIDs, err := s.seenJobIDs(ctx, user.ID)
		if err != nil {
			return nil, err
		}
		if len(seenIDs) > 0 {
			filter["_id"] = bson.M{"$nin": seenIDs}
		}
	}

	opts := options.Find().SetLimit(q.Limit).SetSort(bson.D{{Key: "received_at", Value: -1}})
	cur, err := s.msgs.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var jobs []domain.Job
	if err := cur.All(ctx, &jobs); err != nil {
		return nil, err
	}
	return jobs, nil
}

func (s *Store) GetJob(ctx context.Context, id string) (domain.Job, error) {
	var job domain.Job
	err := s.msgs.FindOne(ctx, bson.M{"_id": id}).Decode(&job)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return domain.Job{}, errors.New("job not found")
		}
		return domain.Job{}, err
	}
	return job, nil
}

func (s *Store) MarkJobSeen(ctx context.Context, userID, jobID string) error {
	if userID == "" || jobID == "" {
		return nil
	}
	_, err := s.seen.UpdateOne(
		ctx,
		bson.M{"user_id": userID, "job_id": jobID},
		bson.M{"$setOnInsert": bson.M{"_id": uuid.NewString(), "user_id": userID, "job_id": jobID, "seen_at": time.Now().UTC()}},
		options.Update().SetUpsert(true),
	)
	return err
}

func (s *Store) EnsureUser(ctx context.Context, in UserUpsertInput) error {
	email := strings.ToLower(strings.TrimSpace(in.Email))
	if email == "" {
		return errors.New("email is required")
	}
	if in.Role == "" {
		in.Role = domain.RoleNormal
	}
	set := bson.M{
		"name":        strings.TrimSpace(in.Name),
		"role":        in.Role,
		"preferences": withDefaultPrefs(in.Preferences),
	}
	if in.PasswordHash != "" {
		set["password_hash"] = in.PasswordHash
	}
	_, err := s.users.UpdateOne(
		ctx,
		bson.M{"email": email},
		bson.M{
			"$set":         set,
			"$setOnInsert": bson.M{"_id": uuid.NewString(), "email": email, "created_at": time.Now().UTC()},
		},
		options.Update().SetUpsert(true),
	)
	return err
}

func (s *Store) FindUserByEmail(ctx context.Context, email string) (domain.User, error) {
	var u domain.User
	err := s.users.FindOne(ctx, bson.M{"email": strings.ToLower(strings.TrimSpace(email))}).Decode(&u)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return domain.User{}, errors.New("invalid credentials")
		}
		return domain.User{}, err
	}
	u.Preferences = withDefaultPrefs(u.Preferences)
	return u, nil
}

func (s *Store) FindUserByID(ctx context.Context, id string) (domain.User, error) {
	var u domain.User
	err := s.users.FindOne(ctx, bson.M{"_id": id}).Decode(&u)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return domain.User{}, errors.New("user not found")
		}
		return domain.User{}, err
	}
	u.Preferences = withDefaultPrefs(u.Preferences)
	return u, nil
}

func (s *Store) ListUsers(ctx context.Context) ([]domain.User, error) {
	cur, err := s.users.Find(ctx, bson.M{}, options.Find().SetSort(bson.D{{Key: "created_at", Value: 1}}))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var users []domain.User
	if err := cur.All(ctx, &users); err != nil {
		return nil, err
	}
	for i := range users {
		users[i].Preferences = withDefaultPrefs(users[i].Preferences)
	}
	return users, nil
}

func (s *Store) UpdateUser(ctx context.Context, id string, role domain.UserRole, prefs *domain.UserPreferences, passwordHash string) error {
	set := bson.M{}
	if role != "" {
		set["role"] = role
	}
	if prefs != nil {
		set["preferences"] = withDefaultPrefs(*prefs)
	}
	if passwordHash != "" {
		set["password_hash"] = passwordHash
	}
	if len(set) == 0 {
		return nil
	}
	_, err := s.users.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": set})
	return err
}

func (s *Store) seenJobIDs(ctx context.Context, userID string) ([]string, error) {
	cur, err := s.seen.Find(ctx, bson.M{"user_id": userID})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var views []domain.SeenJob
	if err := cur.All(ctx, &views); err != nil {
		return nil, err
	}
	out := make([]string, 0, len(views))
	for _, v := range views {
		out = append(out, v.JobID)
	}
	return out, nil
}

func withDefaultPrefs(p domain.UserPreferences) domain.UserPreferences {
	if p.Hours == 0 {
		p.Hours = 24
	}
	return p
}

func regexpEscape(v string) string {
	replacer := strings.NewReplacer(
		`\\`, `\\\\`,
		`.`, `\.`,
		`*`, `\*`,
		`+`, `\+`,
		`?`, `\?`,
		`(`, `\(`,
		`)`, `\)`,
		`[`, `\[`,
		`]`, `\]`,
		`{`, `\{`,
		`}`, `\}`,
		`^`, `\^`,
		`$`, `\$`,
		`|`, `\|`,
	)
	return replacer.Replace(v)
}
