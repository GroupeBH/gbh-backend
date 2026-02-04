package main

import (
	"context"
	"log"
	"time"

	"gbh-backend/internal/config"
	"gbh-backend/internal/db"
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

	log.Println("seed completed")
}
