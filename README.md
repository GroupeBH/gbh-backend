# GBH Backend (Go + MongoDB)

Backend REST JSON pour l’application web **Groupe B-Holding Sarl (GBH)**.

**Fonctionnalités**
- Services (catalogue)
- Disponibilités et créneaux (45 minutes)
- Prise de rendez-vous avec gestion des conflits
- Formulaire de contact
- Simulation d’intention de paiement
- Validation stricte, CORS, rate limiting, logs structurés
- Cache Redis (services + disponibilités)

**Règles métier**
- Lundi–Vendredi : 09h–12h et 14h–17h
- Samedi : 09h–13h
- Dimanche : fermé
- Fuseau horaire : Africa/Kinshasa
- Devise : CDF

## Démarrage rapide

1. Configurer l’environnement
```
cp .env.example .env
```

2. Lancer MongoDB (optionnel via Docker)
```
docker-compose up -d mongo
```

3. Optionnel: lancer Redis (cache)
```
docker-compose up -d redis
```

4. Installer les dépendances et lancer l’API
```
go mod tidy

go run ./cmd/api
```

5. Seed des services
```
go run ./cmd/seed
```

## Endpoints principaux
- `GET /api/services`
- `GET /api/availability?date=YYYY-MM-DD`
- `POST /api/appointments`
- `GET /api/appointments/{id}`
- `POST /api/contact`
- `POST /api/payments/intent`

## OpenAPI
- Fichier: `docs/openapi.yaml`

## Tests
Tests unitaires critiques (disponibilité, créneaux, date passée, conflit).
```
go test ./...
```

## Variables d’environnement
- `APP_ENV`
- `SERVER_ADDR`
- `MONGO_URI`
- `MONGO_DB`
- `FRONTEND_ORIGIN`
- `RATE_LIMIT_APPOINTMENTS`
- `RATE_LIMIT_CONTACT`
- `RATE_LIMIT_WINDOW_SEC`
- `TZ`
- `REDIS_ADDR`
- `REDIS_PASSWORD`
- `REDIS_DB`
- `CACHE_TTL_SECONDS`

## Notes d’implémentation
- Les dates sont stockées en `YYYY-MM-DD` et les heures en `HH:MM`.
- L’unicité des rendez-vous est protégée par un index Mongo `{ date: 1, time: 1 }`.
- Les conflits sont également détectés avant insertion pour fournir une erreur propre.
