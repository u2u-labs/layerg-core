package server

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/u2u-labs/go-layerg-common/runtime"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/gofrs/uuid/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/u2u-labs/go-layerg-common/api"
	"github.com/u2u-labs/go-layerg-common/rtapi"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	NotificationCodeDmRequest        int32 = -1
	NotificationCodeFriendRequest    int32 = -2
	NotificationCodeFriendAccept     int32 = -3
	NotificationCodeGroupAdd         int32 = -4
	NotificationCodeGroupJoinRequest int32 = -5
	NotificationCodeFriendJoinGame   int32 = -6
	NotificationCodeSingleSocket     int32 = -7
	NotificationCodeUserBanned       int32 = -8
)

type notificationCacheableCursor struct {
	NotificationID []byte
	CreateTime     int64
}

func NotificationSend(ctx context.Context, logger *zap.Logger, db *sql.DB, tracker Tracker, messageRouter MessageRouter, notifications map[uuid.UUID][]*api.Notification) error {
	persistentNotifications := make(map[uuid.UUID][]*api.Notification, len(notifications))
	for userID, ns := range notifications {
		for _, userNotification := range ns {
			// Select persistent notifications for storage.
			if userNotification.Persistent {
				if pun := persistentNotifications[userID]; pun == nil {
					persistentNotifications[userID] = []*api.Notification{userNotification}
				} else {
					persistentNotifications[userID] = append(pun, userNotification)
				}
			}
		}
	}

	// Store any persistent notifications.
	if len(persistentNotifications) > 0 {
		if err := NotificationSave(ctx, logger, db, persistentNotifications); err != nil {
			return err
		}
	}

	recipients := make(map[PresenceStream][]*PresenceID, len(notifications))
	for userID := range notifications {
		recipients[PresenceStream{Mode: StreamModeNotifications, Subject: userID}] = make([]*PresenceID, 0, 1)
	}
	tracker.ListPresenceIDByStreams(recipients)

	// Deliver live notifications to connected users.
	for stream, presenceIDs := range recipients {
		if len(presenceIDs) == 0 {
			continue
		}
		ns, found := notifications[stream.Subject]
		if !found {
			continue
		}

		messageRouter.SendToPresenceIDs(logger, presenceIDs, &rtapi.Envelope{
			Message: &rtapi.Envelope_Notifications{
				Notifications: &rtapi.Notifications{
					Notifications: ns,
				},
			},
		}, true)
	}

	return nil
}

func NotificationSendAll(ctx context.Context, logger *zap.Logger, db *sql.DB, gotracker Tracker, messageRouter MessageRouter, notification *api.Notification) error {
	// Non-persistent notifications don't need to work through all database users, just use currently connected notification streams.
	if !notification.Persistent {
		env := &rtapi.Envelope{
			Message: &rtapi.Envelope_Notifications{
				Notifications: &rtapi.Notifications{
					Notifications: []*api.Notification{notification},
				},
			},
		}

		messageRouter.SendToAll(logger, env, true)

		return nil
	}

	const limit = 10_000

	// Start dispatch in paginated batches.
	go func() {
		// Switch to a background context, the caller should not wait for the full operation to complete.
		ctx := context.Background()
		notificationLogger := logger.With(zap.String("notification_subject", notification.Subject))

		var userIDStr string
		for {
			sends := make(map[uuid.UUID][]*api.Notification, limit)

			params := make([]interface{}, 0, 1)
			query := "SELECT id FROM users"
			if userIDStr != "" {
				query += " WHERE id > $1"
				params = append(params, userIDStr)
			}
			query += fmt.Sprintf(" ORDER BY id ASC LIMIT %d", limit)

			rows, err := db.QueryContext(ctx, query, params...)
			if err != nil {
				notificationLogger.Error("Failed to retrieve user data to send notification", zap.Error(err))
				return
			}

			for rows.Next() {
				if err = rows.Scan(&userIDStr); err != nil {
					_ = rows.Close()
					notificationLogger.Error("Failed to scan user data to send notification", zap.String("id", userIDStr), zap.Error(err))
					return
				}
				userID, err := uuid.FromString(userIDStr)
				if err != nil {
					_ = rows.Close()
					notificationLogger.Error("Failed to parse scanned user id data to send notification", zap.String("id", userIDStr), zap.Error(err))
					return
				}
				sends[userID] = []*api.Notification{{
					Id:         uuid.Must(uuid.NewV4()).String(),
					Subject:    notification.Subject,
					Content:    notification.Content,
					Code:       notification.Code,
					SenderId:   notification.SenderId,
					CreateTime: notification.CreateTime,
					Persistent: notification.Persistent,
				}}
			}
			_ = rows.Close()

			if len(sends) == 0 {
				// Pagination finished.
				return
			}

			if err := NotificationSave(ctx, notificationLogger, db, sends); err != nil {
				notificationLogger.Error("Failed to save persistent notifications", zap.Error(err))
				return
			}

			// Deliver live notifications to connected users.
			for userID, notifications := range sends {
				env := &rtapi.Envelope{
					Message: &rtapi.Envelope_Notifications{
						Notifications: &rtapi.Notifications{
							Notifications: []*api.Notification{notifications[0]},
						},
					},
				}

				messageRouter.SendToStream(logger, PresenceStream{Mode: StreamModeNotifications, Subject: userID}, env, true)
			}

			// Stop pagination when reaching the last (incomplete) page.
			if len(sends) < limit {
				return
			}
		}
	}()

	return nil
}

func NotificationList(ctx context.Context, logger *zap.Logger, db *sql.DB, userID uuid.UUID, limit int, cursor string, cacheable bool) (*api.NotificationList, error) {
	var nc *notificationCacheableCursor
	if cursor != "" {
		nc = &notificationCacheableCursor{}
		cb, err := base64.RawURLEncoding.DecodeString(cursor)
		if err != nil {
			logger.Warn("Could not base64 decode notification cursor.", zap.String("cursor", cursor))
			return nil, status.Error(codes.InvalidArgument, "Malformed cursor was used.")
		}
		if err = gob.NewDecoder(bytes.NewReader(cb)).Decode(nc); err != nil {
			logger.Warn("Could not decode notification cursor.", zap.String("cursor", cursor))
			return nil, status.Error(codes.InvalidArgument, "Malformed cursor was used.")
		}
	}

	params := []interface{}{userID}

	limitQuery := ""
	if limit > 0 {
		params = append(params, limit+1)
		limitQuery = " LIMIT $2"
	}

	cursorQuery := ""
	if nc != nil && nc.NotificationID != nil {
		cursorQuery = " AND (user_id, create_time, id) > ($1::UUID, $3::TIMESTAMPTZ, $4::UUID)"
		params = append(params, &pgtype.Timestamptz{Time: time.Unix(0, nc.CreateTime).UTC(), Valid: true}, uuid.FromBytesOrNil(nc.NotificationID))
	}

	query := `
SELECT id, subject, content, code, sender_id, create_time
FROM notification
WHERE user_id = $1` + cursorQuery + `
ORDER BY create_time ASC, id ASC` + limitQuery

	rows, err := db.QueryContext(ctx, query, params...)
	if err != nil {
		logger.Error("Could not retrieve notifications.", zap.Error(err))
		return nil, err
	}

	notifications := make([]*api.Notification, 0, limit)
	var lastCreateTime int64
	var resultCount int
	var hasNextPage bool
	for rows.Next() {
		resultCount++
		if limit > 0 && resultCount > limit {
			hasNextPage = true
			break
		}
		no := &api.Notification{Persistent: true, CreateTime: &timestamppb.Timestamp{}}
		var createTime pgtype.Timestamptz
		if err := rows.Scan(&no.Id, &no.Subject, &no.Content, &no.Code, &no.SenderId, &createTime); err != nil {
			_ = rows.Close()
			logger.Error("Could not scan notification from database.", zap.Error(err))
			return nil, err
		}

		lastCreateTime = createTime.Time.UnixNano()
		no.CreateTime.Seconds = createTime.Time.Unix()
		if no.SenderId == uuid.Nil.String() {
			no.SenderId = ""
		}
		notifications = append(notifications, no)
	}
	_ = rows.Close()

	notificationList := &api.NotificationList{}
	cursorBuf := new(bytes.Buffer)
	if len(notifications) == 0 {
		if cacheable {
			if len(cursor) > 0 {
				notificationList.CacheableCursor = cursor
			} else {
				newCursor := &notificationCacheableCursor{NotificationID: nil, CreateTime: 0}
				if err := gob.NewEncoder(cursorBuf).Encode(newCursor); err != nil {
					logger.Error("Could not create new cursor.", zap.Error(err))
					return nil, err
				}
				notificationList.CacheableCursor = base64.RawURLEncoding.EncodeToString(cursorBuf.Bytes())
			}
		}
	} else {
		notificationList.Notifications = notifications
		if cacheable || hasNextPage {
			lastNotification := notifications[len(notifications)-1]
			newCursor := &notificationCacheableCursor{
				NotificationID: uuid.FromStringOrNil(lastNotification.Id).Bytes(),
				CreateTime:     lastCreateTime,
			}
			if err := gob.NewEncoder(cursorBuf).Encode(newCursor); err != nil {
				logger.Error("Could not create new cursor.", zap.Error(err))
				return nil, err
			}

			notificationList.CacheableCursor = base64.RawURLEncoding.EncodeToString(cursorBuf.Bytes())
		}
	}

	return notificationList, nil
}

func NotificationDelete(ctx context.Context, logger *zap.Logger, db *sql.DB, userID uuid.UUID, notificationIDs []string) error {
	params := []any{userID, notificationIDs}

	query := "DELETE FROM notification WHERE user_id = $1 AND id = ANY($2)"
	logger.Debug("Delete notification query", zap.String("query", query), zap.Any("params", params))
	_, err := db.ExecContext(ctx, query, params...)
	if err != nil {
		logger.Error("Could not delete notifications.", zap.Error(err))
		return err
	}

	return nil
}

func NotificationSave(ctx context.Context, logger *zap.Logger, db *sql.DB, notifications map[uuid.UUID][]*api.Notification) error {
	ids := make([]string, 0, len(notifications))
	userIds := make([]uuid.UUID, 0, len(notifications))
	subjects := make([]string, 0, len(notifications))
	contents := make([]string, 0, len(notifications))
	codes := make([]int32, 0, len(notifications))
	senderIds := make([]string, 0, len(notifications))
	query := `
INSERT INTO
	notification (id, user_id, subject, content, code, sender_id)
SELECT
	unnest($1::uuid[]),
	unnest($2::uuid[]),
	unnest($3::text[]),
	unnest($4::jsonb[]),
	unnest($5::smallint[]),
	unnest($6::uuid[]);
`
	for userID, no := range notifications {
		for _, un := range no {
			ids = append(ids, un.Id)
			userIds = append(userIds, userID)
			subjects = append(subjects, un.Subject)
			contents = append(contents, un.Content)
			codes = append(codes, un.Code)
			senderIds = append(senderIds, un.SenderId)
		}
	}

	if _, err := db.ExecContext(ctx, query, ids, userIds, subjects, contents, codes, senderIds); err != nil {
		logger.Error("Could not save notifications.", zap.Error(err))
		return err
	}

	return nil
}

func NotificationsGetId(ctx context.Context, logger *zap.Logger, db *sql.DB, userID string, ids ...string) ([]*runtime.Notification, error) {
	if len(ids) == 0 {
		return []*runtime.Notification{}, nil
	}

	for _, id := range ids {
		if _, err := uuid.FromString(id); err != nil {
			return nil, errors.New("expects id to be a valid id string")
		}
	}

	params := []any{ids}
	query := "SELECT id, user_id, subject, content, code, sender_id, create_time FROM notification WHERE id = any($1)"
	if userID != "" {
		query += " AND user_id = $2"
		params = append(params, userID)
	}

	rows, err := db.QueryContext(ctx, query, params...)
	if err != nil {
		logger.Error("failed to list notifications by id", zap.Error(err))
		return nil, fmt.Errorf("failed to list notifications by id: %s", err.Error())
	}

	defer rows.Close()

	notifications := make([]*runtime.Notification, 0, len(ids))
	for rows.Next() {
		no := &runtime.Notification{Persistent: true, CreateTime: &timestamppb.Timestamp{}}
		var createTime pgtype.Timestamptz
		var content string
		if err := rows.Scan(&no.Id, &no.UserID, &no.Subject, &content, &no.Code, &no.Sender, &createTime); err != nil {
			_ = rows.Close()
			logger.Error("Failed to scan notification from database.", zap.Error(err))
			return nil, err
		}
		no.CreateTime.Seconds = createTime.Time.Unix()

		var contentMap map[string]any
		if err = json.Unmarshal([]byte(content), &contentMap); err != nil {
			logger.Error("Failed to unmarshal notification content", zap.Error(err))
			return nil, err
		}
		no.Content = contentMap

		if no.Sender == uuid.Nil.String() {
			no.Sender = ""
		}
		notifications = append(notifications, no)
	}

	return notifications, nil
}

func NotificationsDeleteId(ctx context.Context, logger *zap.Logger, db *sql.DB, userID string, ids ...string) error {
	if len(ids) == 0 {
		// NOOP
		return nil
	}

	for _, id := range ids {
		if _, err := uuid.FromString(id); err != nil {
			return errors.New("expects id to be a valid uuid")
		}
	}

	if userID != "" {
		uid, err := uuid.FromString(userID)
		if err != nil {
			return errors.New("expects id to be a valid uuid")
		}

		return NotificationDelete(ctx, logger, db, uid, ids)
	}

	if _, err := db.QueryContext(ctx, "DELETE FROM notification WHERE id = any($1)", ids); err != nil {
		logger.Error("failed to delete notifications by id", zap.Error(err))
		return fmt.Errorf("failed to delete notifications: %s", err.Error())
	}

	return nil
}

type notificationUpdate struct {
	Id      uuid.UUID
	Content map[string]any
	Subject *string
	Sender  *string
}

func NotificationsUpdate(ctx context.Context, logger *zap.Logger, db *sql.DB, updates ...notificationUpdate) error {
	if len(updates) == 0 {
		// NOOP
		return nil
	}

	b := &pgx.Batch{}
	for _, update := range updates {
		b.Queue("UPDATE notification SET content = coalesce($1, content), subject = coalesce($2, subject), sender_id = coalesce($3, sender_id) WHERE id = $4", update.Content, update.Subject, update.Sender, update.Id)
	}

	if err := ExecuteInTxPgx(ctx, db, func(tx pgx.Tx) error {
		r := tx.SendBatch(ctx, b)
		_, err := r.Exec()
		defer r.Close()
		if err != nil {
			return err
		}

		return nil
	}); err != nil {
		logger.Error("failed to update notifications", zap.Error(err))
		return fmt.Errorf("failed to update notifications: %s", err.Error())
	}

	return nil
}
