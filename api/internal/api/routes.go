package api

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5"
	"github.com/ogabrielrodrigues/ama/api/internal/store/pg"
)

const (
	MessageKindMessageCreated = "message_created"
)

type MessageMessageCreated struct {
	ID      string `json:"id"`
	Message string `json:"message"`
}

type Message struct {
	Kind   string `json:"kind"`
	Value  any    `json:"value"`
	RoomID string `json:"-"`
}

func (h apiHandler) NotifyClients(msg Message) {
	h.mut.Lock()
	defer h.mut.Unlock()

	subscribers, ok := h.subscribers[msg.RoomID]
	if !ok || len(subscribers) == 0 {
		return
	}

	for connection, cancel := range subscribers {
		if err := connection.WriteJSON(msg); err != nil {
			slog.Error("failed to send message to client", "error", err)
			cancel()
		}
	}
}

func (h apiHandler) handleSubscribe(w http.ResponseWriter, r *http.Request) {
	raw_room_id := chi.URLParam(r, "room_id")

	room_id, err := uuid.Parse(raw_room_id)
	if err != nil {
		http.Error(w, "invalid room id", http.StatusBadRequest)
		return
	}

	_, err = h.queries.FindRoom(r.Context(), room_id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "room not found", http.StatusBadRequest)
			return
		}

		http.Error(w, "something went wrong", http.StatusInternalServerError)
		return
	}

	connection, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Warn("failed to upgrade connection", "error", err)
		http.Error(w, "failed to upgrade to websocket connection", http.StatusInternalServerError)
		return
	}

	defer connection.Close()

	ctx, cancel := context.WithCancel(r.Context())

	h.mut.Lock()
	if _, ok := h.subscribers[raw_room_id]; !ok {
		h.subscribers[raw_room_id] = make(map[*websocket.Conn]context.CancelFunc)
	}

	slog.Info("new client connected", "room_id", raw_room_id, "client_ip", r.RemoteAddr)
	h.subscribers[raw_room_id][connection] = cancel
	h.mut.Unlock()

	<-ctx.Done()

	h.mut.Lock()
	delete(h.subscribers[raw_room_id], connection)
	h.mut.Unlock()
}

func (h apiHandler) handleCreateRoom(w http.ResponseWriter, r *http.Request) {
	type _body struct {
		Theme string `json:"theme"`
	}

	var body _body
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	room_id, err := h.queries.SaveRoom(r.Context(), body.Theme)
	if err != nil {
		slog.Error("failed to insert room", "error", err)
		http.Error(w, "something went wrong", http.StatusInternalServerError)
		return
	}

	type response struct {
		ID string `json:"id"`
	}

	data, _ := json.Marshal(response{ID: room_id.String()})
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(data)
}

func (h apiHandler) handleFindRooms(w http.ResponseWriter, r *http.Request) {}

func (h apiHandler) handleCreateRoomMessage(w http.ResponseWriter, r *http.Request) {
	raw_room_id := chi.URLParam(r, "room_id")

	room_id, err := uuid.Parse(raw_room_id)
	if err != nil {
		http.Error(w, "invalid room id", http.StatusBadRequest)
		return
	}

	_, err = h.queries.FindRoom(r.Context(), room_id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "room not found", http.StatusBadRequest)
			return
		}

		http.Error(w, "something went wrong", http.StatusInternalServerError)
		return
	}

	type _body struct {
		Message string `json:"message"`
	}

	var body _body
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	message_id, err := h.queries.SaveMessage(r.Context(), pg.SaveMessageParams{
		RoomID:  room_id,
		Message: body.Message,
	})
	if err != nil {
		slog.Error("failed to save message", "error", err)
		http.Error(w, "something went wrong", http.StatusInternalServerError)
		return
	}

	type response struct {
		ID string `json:"id"`
	}

	data, _ := json.Marshal(response{ID: message_id.String()})
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(data)

	go h.NotifyClients(Message{
		Kind:   MessageKindMessageCreated,
		RoomID: raw_room_id,
		Value: MessageMessageCreated{
			ID:      message_id.String(),
			Message: body.Message,
		},
	})
}

func (h apiHandler) handleGetRoomMessages(w http.ResponseWriter, r *http.Request)        {}
func (h apiHandler) handleFindRoomMessage(w http.ResponseWriter, r *http.Request)        {}
func (h apiHandler) handleReactToMessage(w http.ResponseWriter, r *http.Request)         {}
func (h apiHandler) handleRemoveReactFromMessage(w http.ResponseWriter, r *http.Request) {}
func (h apiHandler) handleMarkMessageAsAnswered(w http.ResponseWriter, r *http.Request)  {}
