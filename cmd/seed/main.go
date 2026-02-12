package main

import (
	"context"
	"log"
	"os"
	"time"

	"gbh-backend/internal/auth"
	"gbh-backend/internal/config"
	"gbh-backend/internal/db"
	"gbh-backend/internal/models"
	"gbh-backend/internal/utils"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type seedService struct {
	Name        string
	Description string
	Category    string
	ForAudience string
}

type seedUser struct {
	Username    string
	Email       string
	PasswordEnv string
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, cols, err := db.Connect(ctx, cfg.MongoURI, cfg.MongoDB)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect(context.Background())

	if err := db.EnsureIndexes(ctx, cols); err != nil {
		log.Fatal(err)
	}

	services := []seedService{
		{Name: "Conseil stratégique", Description: "Orientation et vision pour accélérer la croissance.", Category: "Conseil", ForAudience: "organisations"},
		{Name: "Intelligence opérationnelle", Description: "Optimisation des processus et pilotage des activités.", Category: "Opérations", ForAudience: "organisations"},
		{Name: "Laboratoire numérique", Description: "Prototypage et solutions digitales sur mesure.", Category: "Digital", ForAudience: "tous"},
		{Name: "Recrutement", Description: "Sélection et placement des talents adaptés.", Category: "RH", ForAudience: "organisations"},
		{Name: "Formation", Description: "Programmes de montée en compétences ciblés.", Category: "Formation", ForAudience: "tous"},
		{Name: "Fourniture", Description: "Approvisionnement fiable en biens et services.", Category: "Supply", ForAudience: "organisations"},
		{Name: "Entrepreneuriat", Description: "Accompagnement des porteurs de projets.", Category: "Entrepreneuriat", ForAudience: "particuliers"},
		{Name: "Fiscalité", Description: "Conseils fiscaux et conformité locale.", Category: "Finance", ForAudience: "organisations"},
		{Name: "Voyage", Description: "Assistance et organisation de déplacements.", Category: "Voyage", ForAudience: "tous"},
		{Name: "Commission acquisition/vente", Description: "Intermédiation pour acquisition ou vente d'actifs.", Category: "Transactions", ForAudience: "organisations"},
	}

	for _, svc := range services {
		slug := utils.Slugify(svc.Name)
		filter := bson.M{"slug": slug}
		update := bson.M{
			"$setOnInsert": bson.M{
				"_id":         primitive.NewObjectID().Hex(),
				"name":        svc.Name,
				"description": svc.Description,
				"category":    svc.Category,
				"forAudience": svc.ForAudience,
				"slug":        slug,
				"createdAt":   time.Now().In(cfg.Timezone),
			},
		}

		_, err := cols.Services.UpdateOne(ctx, filter, update, options.Update().SetUpsert(true))
		if err != nil {
			log.Fatalf("seed error for %s: %v", svc.Name, err)
		}
	}

	adminUsers := []seedUser{
		{
			Username:    envOrDefault("ADMIN_USER", "admin"),
			Email:       envOrDefault("ADMIN_EMAIL", ""),
			PasswordEnv: "ADMIN_PASSWORD",
		},
		{
			Username:    envOrDefault("ADMIN_USER_2", "admin2"),
			Email:       envOrDefault("ADMIN_EMAIL_2", ""),
			PasswordEnv: "ADMIN_PASSWORD_2",
		},
	}

	for _, admin := range adminUsers {
		password := os.Getenv(admin.PasswordEnv)
		if password == "" {
			log.Printf("seed admin: %s missing, skipping (%s)", admin.Username, admin.PasswordEnv)
			continue
		}
		if err := seedAdminUser(ctx, cols, admin.Username, admin.Email, password, cfg.Timezone); err != nil {
			log.Fatalf("seed admin error for %s: %v", admin.Username, err)
		}
	}

	log.Println("seed completed")
}

func seedAdminUser(ctx context.Context, cols *db.Collections, username, email, password string, loc *time.Location) error {
	if cols == nil || cols.Users == nil {
		return nil
	}
	if username == "" || password == "" {
		return nil
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return err
	}
	now := time.Now().In(loc)
	filter := bson.M{"username": username}
	set := bson.M{
		"passwordHash": hash,
		"role":         models.UserRoleAdmin,
		"updatedAt":    now,
	}
	if email != "" {
		set["email"] = email
	}
	setOnInsert := bson.M{
		"_id":       primitive.NewObjectID().Hex(),
		"username":  username,
		"createdAt": now,
	}
	if email != "" {
		setOnInsert["email"] = email
	}
	update := bson.M{
		"$set":         set,
		"$setOnInsert": setOnInsert,
	}
	_, err = cols.Users.UpdateOne(ctx, filter, update, options.Update().SetUpsert(true))
	return err
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
