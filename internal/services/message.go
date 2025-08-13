package services

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"tomlord.io-backend/internal/database"
	db "tomlord.io-backend/internal/db_sqlc"
)

type MessageService struct {
	dbService database.DBService
}

type MessageInfo struct {
	ID          string  `json:"id"`
	UserID      string  `json:"user_id"`
	UserName    string  `json:"user_name"`
	UserPicture string  `json:"user_picture"`
	PostSlug    string  `json:"post_slug"`
	BlogID      *string `json:"blog_id,omitempty"` // New field for blog reference
	Message     string  `json:"message"`
	ThumbCount  int32   `json:"thumb_count"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
	UserThumbed bool    `json:"user_thumbed,omitempty"`
}

type CreateMessageRequest struct {
	UserID   string `json:"user_id" binding:"required"`
	PostSlug string `json:"post_slug" binding:"required"`
	BlogID   string `json:"blog_id,omitempty"` // Optional blog ID
	Message  string `json:"message" binding:"required"`
}

type UpdateMessageRequest struct {
	MessageID string `json:"message_id" binding:"required"`
	UserID    string `json:"user_id" binding:"required"`
	Message   string `json:"message" binding:"required"`
}

type ListMessagesRequest struct {
	PostSlug string `json:"post_slug,omitempty"` // Made optional
	BlogID   string `json:"blog_id,omitempty"`   // New field for blog-based queries
	BlogSlug string `json:"blog_slug,omitempty"` // New field for blog slug-based queries
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

	var message db.Message
	var err error

	// Fallback to the old method for backward compatibility
	message, err = queries.CreateMessage(ctx, db.CreateMessageParams{
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

// GetMessagesByBlogSlug retrieves messages for a specific blog by blog slug
func (m *MessageService) GetMessagesByBlogSlug(ctx context.Context, req ListMessagesRequest) ([]MessageInfo, error) {
	queries := m.dbService.GetQueries()

	if req.Limit <= 0 {
		req.Limit = 20 // Default limit
	}
	if req.Offset < 0 {
		req.Offset = 0
	}

	messages, err := queries.GetMessagesByBlogSlug(ctx, db.GetMessagesByBlogSlugParams{
		PostSlug: req.BlogSlug,
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
			ThumbCount:  int32(msg.ThumbCount_2), // Use the calculated thumb count from the query
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

// DeleteMessageBySuperUser deletes a message with super user privileges (no ownership check)
func (m *MessageService) DeleteMessageBySuperUser(ctx context.Context, messageID string) error {
	queries := m.dbService.GetQueries()

	messageUUID := pgtype.UUID{}
	if err := messageUUID.Scan(messageID); err != nil {
		return fmt.Errorf("invalid message ID: %w", err)
	}

	err := queries.DeleteMessageBySuperUser(ctx, messageUUID)
	if err != nil {
		return fmt.Errorf("failed to delete message: %w", err)
	}

	return nil
}

// ToggleMessageThumb toggles a thumb/like on a message
func (m *MessageService) ToggleMessageThumb(ctx context.Context, messageID, userID string) (bool, error) {
	// Parse IDs
	messageUUID := pgtype.UUID{}
	if err := messageUUID.Scan(messageID); err != nil {
		return false, fmt.Errorf("invalid message ID: %w", err)
	}

	userUUID := pgtype.UUID{}
	if err := userUUID.Scan(userID); err != nil {
		return false, fmt.Errorf("invalid user ID: %w", err)
	}

	var toggled bool
	if err := m.dbService.WithTx(ctx, func(qtx *db.Queries) error {
		// Check if user has already thumbed this message within the transaction
		t, err := qtx.CheckUserThumbedMessage(ctx, db.CheckUserThumbedMessageParams{
			MessageID: messageUUID,
			UserID:    userUUID,
		})
		if err != nil {
			return fmt.Errorf("failed to check thumb status: %w", err)
		}
		toggled = !t

		if t {
			// Remove thumb
			if err := qtx.DeleteMessageThumb(ctx, db.DeleteMessageThumbParams{
				MessageID: messageUUID,
				UserID:    userUUID,
			}); err != nil {
				return fmt.Errorf("failed to remove thumb: %w", err)
			}
		} else {
			// Add thumb
			if _, err := qtx.CreateMessageThumb(ctx, db.CreateMessageThumbParams{
				MessageID: messageUUID,
				UserID:    userUUID,
			}); err != nil {
				return fmt.Errorf("failed to add thumb: %w", err)
			}
		}

		// Update the thumb_count atomically within the same transaction
		if err := qtx.UpdateMessageThumbCount(ctx, messageUUID); err != nil {
			return fmt.Errorf("failed to update message thumb count: %w", err)
		}
		return nil
	}); err != nil {
		return false, err
	}

	return toggled, nil
}

// GetThumbCount returns the current thumb count for a message
func (s *MessageService) GetThumbCount(ctx context.Context, messageID string) (int32, error) {
	// Parse the message ID
	messageUUID, err := uuid.Parse(messageID)
	if err != nil {
		return 0, fmt.Errorf("invalid message ID: %w", err)
	}

	var pgMessageID pgtype.UUID
	pgMessageID.Bytes = messageUUID
	pgMessageID.Valid = true

	count, err := s.dbService.GetQueries().GetThumbCountForMessage(ctx, pgMessageID)
	if err != nil {
		return 0, fmt.Errorf("failed to get thumb count: %w", err)
	}

	return int32(count), nil
}

// GetMessageByID returns a single message by its ID
func (s *MessageService) GetMessageByID(ctx context.Context, messageID string) (*MessageInfo, error) {
	// Parse the message ID
	messageUUID, err := uuid.Parse(messageID)
	if err != nil {
		return nil, fmt.Errorf("invalid message ID: %w", err)
	}

	var pgMessageID pgtype.UUID
	pgMessageID.Bytes = messageUUID
	pgMessageID.Valid = true

	message, err := s.dbService.GetQueries().GetMessageByID(ctx, pgMessageID)
	if err != nil {
		return nil, fmt.Errorf("failed to get message: %w", err)
	}

	return &MessageInfo{
		ID:          uuid.UUID(message.ID.Bytes).String(),
		UserID:      uuid.UUID(message.UserID.Bytes).String(),
		UserName:    message.UserName,
		UserPicture: message.UserPictureUrl.String,
		PostSlug:    message.PostSlug,
		Message:     message.Message,
		ThumbCount:  int32(message.ThumbCount_2), // Use the dynamically calculated thumb count
		CreatedAt:   message.CreatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   message.UpdatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
	}, nil
}
