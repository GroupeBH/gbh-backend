package handlers

import (
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func normalizeID(doc bson.M) map[string]interface{} {
	if id, ok := doc["_id"]; ok {
		switch v := id.(type) {
		case primitive.ObjectID:
			doc["id"] = v.Hex()
		case string:
			doc["id"] = v
		default:
			doc["id"] = v
		}
		delete(doc, "_id")
	}
	return doc
}
