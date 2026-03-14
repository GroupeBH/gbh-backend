# GBH Backend (Go + MongoDB)

Backend REST JSON pour l’application web **Groupe B-Holding Sarl (GBH)**.

**Fonctionnalités**
- Services (catalogue) avec `shortDescription`, `description`, `benefits`
- Témoignages par service
- Disponibilités et créneaux (45 minutes)
- Prise de rendez-vous avec gestion des conflits
- Recherche de réservation par ID
- Formulaire de contact
- Simulation d’intention de paiement
- Validation stricte, CORS, rate limiting, logs structurés
- Cache Redis (services + disponibilités)
- Endpoints admin protégés (gestion services, créneaux, rendez-vous, contacts)
- Login admin JWT (cookies HttpOnly access + refresh) via la collection `users`

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

2. **Développement** : Lancer avec Docker Compose (MongoDB + Redis locaux)
```
# Avec DB locale
docker-compose --profile localdb up --build

# Sans DB locale (utilise host.docker.internal ou external)
docker-compose up --build
```

3. **Production** : Utiliser Docker Compose prod (MongoDB Atlas + Redis Upstash)
```
# Configurer les variables d'environnement pour prod
export MONGO_URI="mongodb+srv://..."
export REDIS_URL="rediss://..."
# etc.

docker-compose -f docker-compose.prod.yml up --build
```

4. Optionnel: lancer MongoDB/Redis séparément (dev)
```
docker-compose up -d mongo redis
```

5. Installer les dépendances et lancer l’API (sans Docker)
```
go mod tidy
go run ./cmd/api
```

6. Seed des services
```
go run ./cmd/seed
```

## Endpoints principaux
- `GET /api/services`
- `POST /api/services` (admin)
- `PUT /api/services/{id}` (admin)
- `GET /api/services/{id}/availability?date=YYYY-MM-DD&duration=30`
- `GET /api/services/{id}/testimonials`
- `POST /api/services/{id}/testimonials`
- `GET /api/availability?date=YYYY-MM-DD`
- `GET /api/availability/next?from=YYYY-MM-DD&duration=30`
- `POST /api/appointments`
- `GET /api/appointments/{id}`
- `POST /api/appointments/lookup`
- `POST /api/contact`
- `POST /api/payments/intent`

## Endpoints admin
- La plupart des endpoints admin nécessitent `X-Admin-Key` ou un cookie JWT admin valide.
- `POST /api/admin/register` (bootstrap via `ADMIN_SETUP_KEY`)
- `POST /api/admin/login`
- `POST /api/admin/refresh`
- `POST /api/admin/logout`
- `POST /api/admin/services`
- `PUT /api/admin/services/{id}`
- `DELETE /api/admin/services/{id}`
- `POST /api/admin/blocks`
- `DELETE /api/admin/blocks/{id}`
- `POST /api/admin/users`
- `PATCH /api/admin/users/{id}/password`
- `GET /api/admin/appointments?date=YYYY-MM-DD`
- `PATCH /api/admin/appointments/{id}/status`
- `GET /api/admin/contacts`

## OpenAPI
- Fichier: `docs/openapi.yaml`

## Docker

### Développement
Utilise `docker-compose.yml` avec MongoDB et Redis locaux :
```
# Avec DB locale
docker-compose --profile localdb up --build

# Sans DB locale (utilise host.docker.internal ou external)
docker-compose up --build
```
- Hot reload avec Air
- Base de données locale persistée (profil `localdb`)
- Cache Redis local avec persistence
- Healthchecks pour services
- Variables d'env depuis `.env` + overrides

**Note** : Sans profil `localdb`, définis `MONGO_URI_DOCKER` dans `.env` pour pointer vers une DB externe :
```
MONGO_URI_DOCKER=mongodb://host.docker.internal:27017/gbh
```

### Production
Utilise `docker-compose.prod.yml` sans services locaux (pointe vers Atlas/Upstash) :
```
docker-compose -f docker-compose.prod.yml up --build
```
- Image optimisée (multi-stage build)
- Variables d'environnement externes requises
- Pas de volumes locaux pour DB/cache

### Variables pour Production
Dans `docker-compose.prod.yml`, configure :
- `MONGO_URI` : URI MongoDB Atlas
- `REDIS_URL` : URL Redis Upstash
- `FIREBASE_CREDENTIALS_BASE64` : Credentials Firebase encodés
- Autres variables sensibles...

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
- `REDIS_URL`
- `REDIS_ADDR`
- `REDIS_PASSWORD`
- `REDIS_DB`
- `CACHE_TTL_SECONDS`
- `ADMIN_API_KEY`
- `ADMIN_SETUP_KEY` (clé requise pour `POST /api/admin/register`)
- `ADMIN_USER` (seed admin)
- `ADMIN_EMAIL` (email admin optionnel)
- `ADMIN_PASSWORD` (seed admin)
- `ADMIN_USER_2` (seed admin optionnel)
- `ADMIN_EMAIL_2` (email admin optionnel)
- `ADMIN_PASSWORD_2` (seed admin optionnel)
- `JWT_SECRET`
- `ACCESS_TTL_MINUTES`
- `REFRESH_TTL_MINUTES`
- `COOKIE_SECURE`
- `BREVO_API_KEY`
- `BREVO_SENDER_EMAIL`
- `BREVO_SENDER_NAME`
- `BREVO_SANDBOX`
- `FIREBASE_CREDENTIALS_FILE` (ou `GOOGLE_APPLICATION_CREDENTIALS`)
- `FIREBASE_CREDENTIALS_BASE64` (contenu JSON encodé en base64, prend priorité sur le fichier)

## Notes d’implémentation
- Les dates sont stockées en `YYYY-MM-DD` et les heures en `HH:MM`.
- L’unicité des rendez-vous est protégée par un index Mongo `{ date: 1, time: 1 }`.
- Les conflits sont également détectés avant insertion pour fournir une erreur propre.
- Les disponibilités acceptent un paramètre `duration` (multiple de 15 minutes). Par défaut: 45 minutes.
- La création de rendez-vous accepte `duration` (multiple de 15 minutes). Par défaut: 45 minutes.
- `POST /api/appointments` renvoie aussi `availableSlots` (créneaux restants pour la date/durée demandées).
