-- name: FindRoom :one
SELECT
  "id", "theme"
FROM rooms
WHERE id = $1;

-- name: FindRooms :many
SELECT
  "id", "theme"
FROM rooms;

-- name: SaveRoom :one
INSERT INTO rooms
  ( "theme" ) VALUES
  ( $1 )
RETURNING "id";

-- name: FindMessage :one
SELECT
  "id", "room_id", "message", "reaction_count", "answered"
FROM messages
WHERE
  id = $1;

-- name: FindRoomMessages :many
SELECT
  "id", "room_id", "message", "reaction_count", "answered"
FROM messages
WHERE
  room_id = $1;

-- name: SaveMessage :one
INSERT INTO messages
  ( "room_id", "message" ) VALUES
  ( $1, $2 )
RETURNING "id";

-- name: ReactToMessage :one
UPDATE messages
SET
  reaction_count = reaction_count + 1
WHERE
  id = $1
RETURNING reaction_count;

-- name: RemoveReactionFromMessage :one
UPDATE messages
SET
  reaction_count = reaction_count - 1
WHERE
  id = $1
RETURNING reaction_count;

-- name: MarkMessageAsAnswered :exec
UPDATE messages
SET
  answered = true
WHERE
  id = $1;