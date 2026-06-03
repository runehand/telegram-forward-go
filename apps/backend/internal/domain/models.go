package domain

import "time"

type JobMessage struct {
	ID          string    `bson:"_id,omitempty" json:"id"`
	TelegramMsg int       `bson:"telegram_msg_id" json:"telegram_msg_id"`
	SenderID    int64     `bson:"sender_id" json:"sender_id"`
	PeerID      string    `bson:"peer_id" json:"peer_id"`
	Text        string    `bson:"text" json:"text"`
	JobLink     string    `bson:"job_link" json:"job_link"`
	ReceivedAt  time.Time `bson:"received_at" json:"received_at"`
}

type User struct {
	ID           string `bson:"_id,omitempty" json:"id"`
	Email        string `bson:"email" json:"email"`
	PasswordHash string `bson:"password_hash" json:"-"`
}
