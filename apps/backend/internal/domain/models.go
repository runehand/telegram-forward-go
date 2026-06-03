package domain

import "time"

type UserRole string

const (
	RoleAdmin  UserRole = "admin"
	RoleNormal UserRole = "normal"
)

type Job struct {
	ID              string    `bson:"_id" json:"id"`
	TelegramMsgID   int       `bson:"telegram_msg_id" json:"telegram_msg_id"`
	SenderID        int64     `bson:"sender_id" json:"sender_id"`
	PeerID          string    `bson:"peer_id" json:"peer_id"`
	Title           string    `bson:"title" json:"title"`
	RawText         string    `bson:"raw_text" json:"raw_text"`
	JobLink         string    `bson:"job_link" json:"job_link"`
	Skills          []string  `bson:"skills" json:"skills"`
	Tags            []string  `bson:"tags" json:"tags"`
	SpecialTags     []string  `bson:"special_tags" json:"special_tags"`
	Description     string    `bson:"description" json:"description"`
	Questions       []string  `bson:"questions" json:"questions"`
	ClientLocation  string    `bson:"client_location" json:"client_location"`
	Countries       []string  `bson:"countries" json:"countries"`
	Budget          string    `bson:"budget" json:"budget"`
	ProjectType     string    `bson:"project_type" json:"project_type"`
	ExperienceLevel string    `bson:"experience_level" json:"experience_level"`
	Category        string    `bson:"category" json:"category"`
	Subcategory     string    `bson:"subcategory" json:"subcategory"`
	PaymentVerified bool      `bson:"payment_verified" json:"payment_verified"`
	OnlyUS          bool      `bson:"only_us" json:"only_us"`
	OnlyMobile      bool      `bson:"only_mobile" json:"only_mobile"`
	OnlyCountry     bool      `bson:"only_country" json:"only_country"`
	PostedAgoText   string    `bson:"posted_ago_text" json:"posted_ago_text"`
	ReceivedAt      time.Time `bson:"received_at" json:"received_at"`
}

type JobQuery struct {
	Limit      int64
	Query      string
	OnlyUnseen bool
	OnlyUS     bool
	OnlyMobile bool
	Verified   *bool
	Country    string
	Tag        string
	Hours      int
}

type User struct {
	ID           string          `bson:"_id" json:"id"`
	Name         string          `bson:"name" json:"name"`
	Email        string          `bson:"email" json:"email"`
	Role         UserRole        `bson:"role" json:"role"`
	PasswordHash string          `bson:"password_hash" json:"-"`
	Preferences  UserPreferences `bson:"preferences" json:"preferences"`
	CreatedAt    time.Time       `bson:"created_at" json:"created_at"`
}

type UserPreferences struct {
	OnlyUnseen bool   `bson:"only_unseen" json:"only_unseen"`
	OnlyUS     bool   `bson:"only_us" json:"only_us"`
	OnlyMobile bool   `bson:"only_mobile" json:"only_mobile"`
	Country    string `bson:"country" json:"country"`
	Hours      int    `bson:"hours" json:"hours"`
}

type SeenJob struct {
	ID     string    `bson:"_id" json:"id"`
	UserID string    `bson:"user_id" json:"user_id"`
	JobID  string    `bson:"job_id" json:"job_id"`
	SeenAt time.Time `bson:"seen_at" json:"seen_at"`
}
