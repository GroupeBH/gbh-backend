package handlers

import (
	"context"
	"net/http"
	"time"

	"gbh-backend/internal/transport"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (s *Server) GetServices(w http.ResponseWriter, r *http.Request) {
	cacheKey := "services:all"
	if s.Cache != nil {
		if cached, ok, err := s.Cache.Get(r.Context(), cacheKey); err == nil && ok {
			writeCachedJSON(w, http.StatusOK, cached)
			return
		}
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	cursor, err := s.Cols.Services.Find(ctx, bson.D{}, options.Find().SetSort(bson.D{{Key: "name", Value: 1}}))
	if err != nil {
		transport.WriteError(w, http.StatusInternalServerError, "database error", nil)
		return
	}
	defer cursor.Close(ctx)

	var items []map[string]interface{}
	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			transport.WriteError(w, http.StatusInternalServerError, "decode error", nil)
			return
		}
		items = append(items, normalizeID(doc))
	}
	if err := cursor.Err(); err != nil {
		transport.WriteError(w, http.StatusInternalServerError, "cursor error", nil)
		return
	}

	response := map[string]interface{}{
		"services": items,
	}

	if payload, err := encodeJSON(response); err == nil && s.Cache != nil {
		_ = s.Cache.Set(r.Context(), cacheKey, payload, time.Duration(s.Cfg.CacheTTLSeconds)*time.Second)
	}

	transport.WriteJSON(w, http.StatusOK, response)
}
