package services

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"tomlord.io-backend/internal/database"
	"tomlord.io-backend/internal/db"
)

type MessageService struct {
	dbService database.DBService
}

type MessageInfo struct {
	ID          string `json:"id"`
	UserID      string `json:"user_id"`
	UserName    string `json:"user_name"`
	UserPicture string `json:"user_picture"`
	PostSlug    string `json:"post_slug"`
	Message     string `json:"message"`
	ThumbCount  int32  `json:"thumb_count"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
	UserThumbed bool   `json:"user_thumbed,omitempty"`
}

type CreateMessageRequest struct {
	UserID   string `json:"user_id" binding:"required"`
	PostSlug string `json:"post_slug" binding:"required"`
	Message  string `json:"message" binding:"required"`
}

type UpdateMessageRequest struct {
	MessageID string `json:"message_id" binding:"required"`
	UserID    string `json:"user_id" binding:"required"`
	Message   string `json:"message" binding:"required"`
}

type ListMessagesRequest struct {
	PostSlug string `json:"post_slug" binding:"required"`
	Limit    int32  `json:"limit"`
	Offset   int32  `json:"offset"`
	UserID   string `json:"user_id,omitempty"` // For checking if user has thumbed
}

// NewMessageService creates a new message service
func NewMessageService(dbService database.DBService) *MessageService {
	return &MessageService{
		dbService: dbService,
	}
}

// CreateMessage creates a new message/comment
func (m *MessageService) CreateMessage(ctx context.Context, req CreateMessageRequest) (*MessageInfo, error) {
	queries := m.dbService.GetQueries()

	userUUID := pgtype.UUID{}
	if err := userUUID.Scan(req.UserID); err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	message, err := queries.CreateMessage(ctx, db.CreateMessageParams{
		UserID:   userUUID,
		PostSlug: req.PostSlug,
		Message:  req.Message,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create message: %w", err)
	}

	// Get the message with user info
	messageWithUser, err := queries.GetMessageByID(ctx, message.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get created message: %w", err)
	}

	return &MessageInfo{
		ID:          uuid.UUID(message.ID.Bytes).String(),
		UserID:      uuid.UUID(message.UserID.Bytes).String(),
		UserName:    messageWithUser.UserName,
		UserPicture: messageWithUser.UserPictureUrl.String,
		PostSlug:    message.PostSlug,
		Message:     message.Message,
		ThumbCount:  message.ThumbCount.Int32,
		CreatedAt:   message.CreatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   message.UpdatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
	}, nil
}

// GetMessagesByPostSlug retrieves messages for a specific blog post
func (m *MessageService) GetMessagesByPostSlug(ctx context.Context, req ListMessagesRequest) ([]MessageInfo, error) {
	queries := m.dbService.GetQueries()

	if req.Limit <= 0 {
		req.Limit = 20 // Default limit
	}
	if req.Offset < 0 {
		req.Offset = 0
	}

	messages, err := queries.GetMessagesByPostSlug(ctx, db.GetMessagesByPostSlugParams{
		PostSlug: req.PostSlug,
		Limit:    req.Limit,
		Offset:   req.Offset,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get messages: %w", err)
	}

	result := make([]MessageInfo, len(messages))
	for i, msg := range messages {
		result[i] = MessageInfo{
			ID:          uuid.UUID(msg.ID.Bytes).String(),
			UserID:      uuid.UUID(msg.UserID.Bytes).String(),
			UserName:    msg.UserName,
			UserPicture: msg.UserPictureUrl.String,
			PostSlug:    msg.PostSlug,
			Message:     msg.Message,
			ThumbCount:  msg.ThumbCount.Int32,
			CreatedAt:   msg.CreatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt:   msg.UpdatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
		}

		// Check if the requesting user has thumbed this message
		if req.UserID != "" {
			userUUID := pgtype.UUID{}
			if err := userUUID.Scan(req.UserID); err == nil {
				messageUUID := pgtype.UUID{}
				if err := messageUUID.Scan(uuid.UUID(msg.ID.Bytes).String()); err == nil {
					thumbed, err := queries.CheckUserThumbedMessage(ctx, db.CheckUserThumbedMessageParams{
						MessageID: messageUUID,
						UserID:    userUUID,
					})
					if err == nil {
						result[i].UserThumbed = thumbed
					}
				}
			}
		}
	}

	return result, nil
}

// UpdateMessage updates an existing message
func (m *MessageService) UpdateMessage(ctx context.Context, req UpdateMessageRequest) (*MessageInfo, error) {
	queries := m.dbService.GetQueries()

	messageUUID := pgtype.UUID{}
	if err := messageUUID.Scan(req.MessageID); err != nil {
		return nil, fmt.Errorf("invalid message ID: %w", err)
	}

	userUUID := pgtype.UUID{}
	if err := userUUID.Scan(req.UserID); err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	message, err := queries.UpdateMessage(ctx, db.UpdateMessageParams{
		ID:      messageUUID,
		Message: req.Message,
		UserID:  userUUID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update message: %w", err)
	}

	// Get the message with user info
	messageWithUser, err := queries.GetMessageByID(ctx, message.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get updated message: %w", err)
	}

	return &MessageInfo{
		ID:          uuid.UUID(message.ID.Bytes).String(),
		UserID:      uuid.UUID(message.UserID.Bytes).String(),
		UserName:    messageWithUser.UserName,
		UserPicture: messageWithUser.UserPictureUrl.String,
		PostSlug:    message.PostSlug,
		Message:     message.Message,
		ThumbCount:  message.ThumbCount.Int32,
		CreatedAt:   message.CreatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   message.UpdatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
	}, nil
}

// DeleteMessage deletes a message
func (m *MessageService) DeleteMessage(ctx context.Context, messageID, userID string) error {
	queries := m.dbService.GetQueries()

	messageUUID := pgtype.UUID{}
	if err := messageUUID.Scan(messageID); err != nil {
		return fmt.Errorf("invalid message ID: %w", err)
	}

	userUUID := pgtype.UUID{}
	if err := userUUID.Scan(userID); err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}

	err := queries.DeleteMessage(ctx, db.DeleteMessageParams{
		ID:     messageUUID,
		UserID: userUUID,
	})
	if err != nil {
		return fmt.Errorf("failed to delete message: %w", err)
	}

	return nil
}

// ToggleMessageThumb toggles a thumb/like on a message
func (m *MessageService) ToggleMessageThumb(ctx context.Context, messageID, userID string) (bool, error) {
	queries := m.dbService.GetQueries()

	messageUUID := pgtype.UUID{}
	if err := messageUUID.Scan(messageID); err != nil {
		return false, fmt.Errorf("invalid message ID: %w", err)
	}

	userUUID := pgtype.UUID{}
	if err := userUUID.Scan(userID); err != nil {
		return false, fmt.Errorf("invalid user ID: %w", err)
	}

	// Check if user has already thumbed this message
	thumbed, err := queries.CheckUserThumbedMessage(ctx, db.CheckUserThumbedMessageParams{
		MessageID: messageUUID,
		UserID:    userUUID,
	})
	if err != nil {
		return false, fmt.Errorf("failed to check thumb status: %w", err)
	}

	if thumbed {
		// Remove thumb
		err = queries.DeleteMessageThumb(ctx, db.DeleteMessageThumbParams{
			MessageID: messageUUID,
			UserID:    userUUID,
		})
		if err != nil {
			return false, fmt.Errorf("failed to remove thumb: %w", err)
		}
		return false, nil
	} else {
		// Add thumb
		_, err = queries.CreateMessageThumb(ctx, db.CreateMessageThumbParams{
			MessageID: messageUUID,
			UserID:    userUUID,
		})
		if err != nil {
			return false, fmt.Errorf("failed to add thumb: %w", err)
		}
		return true, nil
	}
}

// GetThumbCount gets the thumb count for a message
func (m *MessageService) GetThumbCount(ctx context.Context, messageID string) (int64, error) {
	queries := m.dbService.GetQueries()

	messageUUID := pgtype.UUID{}
	if err := messageUUID.Scan(messageID); err != nil {
		return 0, fmt.Errorf("invalid message ID: %w", err)
	}

	count, err := queries.GetThumbCountForMessage(ctx, messageUUID)
	if err != nil {
		return 0, fmt.Errorf("failed to get thumb count: %w", err)
	}

	return count, nil
}
