package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"
	"unicode"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/surrealdb/surrealdb.go"
	"github.com/surrealdb/surrealdb.go/pkg/models"
)

// Config holds basic sizing for the sample dataset.
type Config struct {
	Namespace         string
	Database          string
	NumUsers          int
	NumProducts       int
	NumOrders         int
	NumPosts          int
	NumKV             int
	NumMetrics        int
	MaxFollowsPerUser int
}

// Relational model
type User struct {
	ID        *models.RecordID `json:"id,omitempty"`
	Name      string           `json:"name"`
	Email     string           `json:"email"`
	Age       int              `json:"age"`
	CreatedAt time.Time        `json:"created_at"`
}

type Product struct {
	ID          *models.RecordID `json:"id,omitempty"`
	Name        string           `json:"name"`
	SKU         string           `json:"sku"`
	Description string           `json:"description"`
	PriceCents  int              `json:"price_cents"`
	CreatedAt   time.Time        `json:"created_at"`
}

type Order struct {
	ID         *models.RecordID `json:"id,omitempty"`
	User       *models.RecordID `json:"user"`
	Status     string           `json:"status"`
	CreatedAt  time.Time        `json:"created_at"`
	TotalCents int              `json:"total_cents"`
}

type OrderItem struct {
	ID         *models.RecordID `json:"id,omitempty"`
	Order      *models.RecordID `json:"order"`
	Product    *models.RecordID `json:"product"`
	Quantity   int              `json:"quantity"`
	PriceCents int              `json:"price_cents"`
}

// Document model
type Comment struct {
	Author   string    `json:"author"`
	Email    string    `json:"email"`
	Body     string    `json:"body"`
	PostedAt time.Time `json:"posted_at"`
}

type Post struct {
	ID          *models.RecordID `json:"id,omitempty"`
	Title       string           `json:"title"`
	Body        string           `json:"body"`
	Tags        []string         `json:"tags"`
	Metadata    map[string]any   `json:"metadata"`
	Comments    []Comment        `json:"comments"`
	PublishedAt time.Time        `json:"published_at"`
}

// Time-series model
type Metric struct {
	ID    *models.RecordID `json:"id,omitempty"`
	Name  string           `json:"name"`
	TS    time.Time        `json:"ts"`
	Value float64          `json:"value"`
}

func main() {
	gofakeit.Seed(time.Now().UnixNano())

	cfg := Config{
		NumUsers:          50,
		NumProducts:       20,
		NumOrders:         80,
		NumPosts:          20,
		NumKV:             20,
		NumMetrics:        24 * 5, // 5 hours of 1-min metrics
		MaxFollowsPerUser: 5,
	}

	// Generate fake namespace and database names (overridable via env vars)
	cfg.Namespace = getenv("SURREALDB_NAMESPACE", "ns_sample" /*randomIdent("ns")*/)
	cfg.Database = getenv("SURREALDB_DATABASE", "ns_sample" /*randomIdent("db")*/)

	url := getenv("SURREALDB_URL", "ws://localhost:8000")
	user := getenv("SURREALDB_USER", "root")
	pass := getenv("SURREALDB_PASS", "SecurePassword123!")

	ctx := context.Background()

	db, err := surrealdb.FromEndpointURLString(ctx, url)
	if err != nil {
		log.Fatalf("connect: %v", err)
	}

	defer func(db *surrealdb.DB, ctx context.Context) {
		closeErr := db.Close(ctx)
		if closeErr != nil {

		}
	}(db, ctx)

	token, err := db.SignIn(ctx, surrealdb.Auth{
		Username: user,
		Password: pass,
	})
	if err != nil {
		log.Fatalf("sign in: %v", err)
	}
	if err := db.Authenticate(ctx, token); err != nil {
		log.Fatalf("authenticate: %v", err)
	}

	if err := ensureNamespaceAndDatabase(ctx, db, cfg.Namespace, cfg.Database); err != nil {
		log.Fatalf("ensure namespace/database: %v", err)
	}

	if err := db.Use(ctx, cfg.Namespace, cfg.Database); err != nil {
		log.Fatalf("use ns/db: %v", err)
	}

	log.Printf("Seeding SurrealDB ns=%q db=%q", cfg.Namespace, cfg.Database)

	users, products, err := seedRelational(ctx, db, cfg)
	if err != nil {
		log.Fatalf("seed relational: %v", err)
	}

	log.Printf("Relational seeding complete: %d users, %d products", len(users), len(products))

	if err := seedDocuments(ctx, db, cfg); err != nil {
		log.Fatalf("seed documents: %v", err)
	}

	if err := seedKeyValues(ctx, db, cfg); err != nil {
		log.Fatalf("seed key-value: %v", err)
	}

	if err := seedGraph(ctx, db, cfg, users); err != nil {
		log.Fatalf("seed graph: %v", err)
	}

	if err := seedMetrics(ctx, db, cfg); err != nil {
		log.Fatalf("seed metrics: %v", err)
	}

	log.Println("Done.")
}

// ensureNamespaceAndDatabase creates namespace+database if needed.
func ensureNamespaceAndDatabase(ctx context.Context, db *surrealdb.DB, ns, database string) error {
	sql := fmt.Sprintf(`
DEFINE NAMESPACE IF NOT EXISTS %s;
USE NS %s;
DEFINE DATABASE IF NOT EXISTS %s;
`, ns, ns, database)

	if _, err := surrealdb.Query[any](ctx, db, sql, nil); err != nil {
		return fmt.Errorf("define namespace/database: %w", err)
	}
	return nil
}

// --- Relational seeding (users, products, orders, order_items) ---

func seedRelational(ctx context.Context, db *surrealdb.DB, cfg Config) ([]User, []Product, error) {
	users := make([]User, 0, cfg.NumUsers)
	products := make([]Product, 0, cfg.NumProducts)
	orders := make([]Order, 0, cfg.NumOrders)

	// Users
	for i := 0; i < cfg.NumUsers; i++ {
		u := fakeUser()
		created, err := surrealdb.Create[User](ctx, db, models.Table("user"), u)
		if err != nil {
			return nil, nil, fmt.Errorf("create user: %w", err)
		}
		users = append(users, *created)
	}
	log.Printf("inserted %d users", len(users))

	// Products
	for i := 0; i < cfg.NumProducts; i++ {
		p := fakeProduct()
		created, err := surrealdb.Create[Product](ctx, db, models.Table("product"), p)
		if err != nil {
			return nil, nil, fmt.Errorf("create product: %w", err)
		}
		products = append(products, *created)
	}
	log.Printf("inserted %d products", len(products))

	if len(users) == 0 || len(products) == 0 {
		return users, products, nil
	}

	// Orders + items
	for i := 0; i < cfg.NumOrders; i++ {
		customer := users[rand.Intn(len(users))]
		order := Order{
			User:       customer.ID,
			Status:     randomStatus(),
			CreatedAt:  randomTimeWithinDays(60),
			TotalCents: 0,
		}

		createdOrder, err := surrealdb.Create[Order](ctx, db, models.Table("order"), order)
		if err != nil {
			return nil, nil, fmt.Errorf("create order: %w", err)
		}

		total := 0
		numItems := rand.Intn(3) + 1
		for j := 0; j < numItems; j++ {
			product := products[rand.Intn(len(products))]
			qty := rand.Intn(3) + 1
			lineTotal := qty * product.PriceCents
			total += lineTotal

			item := OrderItem{
				Order:      createdOrder.ID,
				Product:    product.ID,
				Quantity:   qty,
				PriceCents: lineTotal,
			}
			if _, err := surrealdb.Create[OrderItem](ctx, db, models.Table("order_item"), item); err != nil {
				return nil, nil, fmt.Errorf("create order_item: %w", err)
			}
		}

		// Update order total
		if createdOrder.ID != nil {
			if _, err := surrealdb.Update[Order](ctx, db, *createdOrder.ID, map[string]any{
				"total_cents": total,
			}); err != nil {
				return nil, nil, fmt.Errorf("update order total: %w", err)
			}
		}

		orders = append(orders, *createdOrder)
	}
	log.Printf("inserted %d orders with items", len(orders))

	return users, products, nil
}

func fakeUser() User {
	return User{
		Name:      gofakeit.Name(),
		Email:     gofakeit.Email(),
		Age:       gofakeit.Number(18, 68),
		CreatedAt: randomTimeWithinDays(365),
	}
}

func fakeProduct() Product {
	return Product{
		Name:        gofakeit.Product().Name,
		SKU:         fmt.Sprintf("SKU-%06d", rand.Intn(1000000)),
		Description: gofakeit.Sentence(12),
		PriceCents:  (rand.Intn(490) + 10) * 100, // $10–$500
		CreatedAt:   randomTimeWithinDays(180),
	}
}

// --- Document-style posts ---

func seedDocuments(ctx context.Context, db *surrealdb.DB, cfg Config) error {
	tagPool := []string{
		"go", "surrealdb", "prometheus", "graphql",
		"performance", "cloud", "testing", "database",
	}

	for i := 0; i < cfg.NumPosts; i++ {
		p := Post{
			Title:       fmt.Sprintf("Sample post #%d", i+1),
			Body:        randomParagraph(),
			PublishedAt: randomTimeWithinDays(90),
			Metadata: map[string]any{
				"category": randomChoice([]string{"tech", "howto", "opinion", "news"}),
				"lang":     "en",
				"featured": rand.Intn(4) == 0,
			},
		}

		numTags := rand.Intn(3) + 1
		p.Tags = randomTags(tagPool, numTags)

		// Comments
		nComments := rand.Intn(4) // 0–3 comments
		for j := 0; j < nComments; j++ {
			c := Comment{
				Author:   gofakeit.Name(),
				Email:    gofakeit.Email(),
				Body:     gofakeit.Sentence(15),
				PostedAt: p.PublishedAt.Add(time.Duration(rand.Intn(72)) * time.Hour),
			}
			p.Comments = append(p.Comments, c)
		}

		if _, err := surrealdb.Create[Post](ctx, db, models.Table("post"), p); err != nil {
			return fmt.Errorf("create post: %w", err)
		}
	}

	log.Printf("inserted %d posts (document-style)", cfg.NumPosts)
	return nil
}

// --- Key-value style data ---

func seedKeyValues(ctx context.Context, db *surrealdb.DB, cfg Config) error {
	baseKeys := []string{
		"site_name",
		"max_users",
		"welcome_message",
		"feature_flag_new_ui",
		"default_locale",
	}

	for i := 0; i < cfg.NumKV; i++ {
		// Respect cancellation if caller wants to stop seeding.
		if err := ctx.Err(); err != nil {
			return err
		}

		var key string
		if i < len(baseKeys) {
			key = baseKeys[i]
		} else {
			key = fmt.Sprintf("setting_%d", i-len(baseKeys)+1)
		}

		value := gofakeit.BuzzWord()
		if value == "" {
			value = fmt.Sprintf("value-%d", rand.Intn(1_000_000))
		}

		rid := models.NewRecordID("kv", key)

		// UPSERT is idempotent: creates if missing, overwrites if already there.
		if _, err := surrealdb.Upsert[map[string]any](ctx, db, rid, map[string]any{
			"value": value,
		}); err != nil {
			return fmt.Errorf("upsert kv %q: %w", key, err)
		}
	}

	log.Printf("inserted/updated %d key-value pairs", cfg.NumKV)
	return nil
}

// --- Graph-style data: user ->follows-> user ---

func seedGraph(ctx context.Context, db *surrealdb.DB, cfg Config, users []User) error {
	if len(users) < 2 {
		return nil
	}

	for _, u := range users {
		if u.ID == nil {
			continue
		}
		numFollows := rand.Intn(cfg.MaxFollowsPerUser + 1)
		for j := 0; j < numFollows; j++ {
			v := users[rand.Intn(len(users))]
			if v.ID == nil {
				continue
			}
			// avoid self-follow
			if u.ID.Table == v.ID.Table && u.ID.ID == v.ID.ID {
				continue
			}

			rel := &surrealdb.Relationship{
				In:       *u.ID,
				Out:      *v.ID,
				Relation: models.Table("follows"),
				Data: map[string]any{
					"since": time.Now().Add(-time.Duration(rand.Intn(365*24)) * time.Hour),
				},
			}

			if _, err := surrealdb.Relate[map[string]any](ctx, db, rel); err != nil {
				return fmt.Errorf("relate %s -> %s: %w", u.ID, v.ID, err)
			}
		}
	}

	log.Printf("inserted follow relationships (graph edges) for %d users", len(users))
	return nil
}

// --- Time-series / metrics data ---

func seedMetrics(ctx context.Context, db *surrealdb.DB, cfg Config) error {
	names := []string{
		"api_latency_ms",
		"worker_jobs_in_queue",
		"worker_jobs_processed",
		"frontend_page_views",
	}

	now := time.Now()
	count := 0

	for i := 0; i < cfg.NumMetrics; i++ {
		ts := now.Add(-time.Duration(i) * time.Minute)
		name := names[rand.Intn(len(names))]

		value := rand.Float64() * 1000
		if name == "worker_jobs_in_queue" {
			value = rand.Float64() * 200
		}
		if name == "worker_jobs_processed" {
			value = rand.Float64() * 500
		}

		m := Metric{
			Name:  name,
			TS:    ts,
			Value: value,
		}

		if _, err := surrealdb.Create[Metric](ctx, db, models.Table("metric"), m); err != nil {
			return fmt.Errorf("create metric: %w", err)
		}
		count++
	}

	log.Printf("inserted %d metrics", count)
	return nil
}

// --- Helpers ---

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// Generate a Surreal-safe identifier like "ns_forest" or "db_river".
func randomIdent(prefix string) string {
	word := gofakeit.Word() // simple single word
	word = strings.ToLower(word)

	var cleaned []rune
	for _, r := range word {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			cleaned = append(cleaned, r)
		} else {
			cleaned = append(cleaned, '_')
		}
	}
	if len(cleaned) == 0 {
		cleaned = []rune("default")
	}
	// Surreal identifiers can't start with a digit
	if unicode.IsDigit(cleaned[0]) {
		cleaned = append([]rune{'x'}, cleaned...)
	}
	return fmt.Sprintf("%s_%s", prefix, string(cleaned))
}

func randomTimeWithinDays(days int) time.Time {
	hours := rand.Intn(days * 24)
	return time.Now().Add(-time.Duration(hours) * time.Hour)
}

func randomStatus() string {
	statuses := []string{
		"pending",
		"paid",
		"processing",
		"shipped",
		"completed",
		"cancelled",
	}
	return statuses[rand.Intn(len(statuses))]
}

func randomChoice(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[rand.Intn(len(values))]
}

func randomTags(pool []string, n int) []string {
	if n <= 0 || len(pool) == 0 {
		return nil
	}
	if n > len(pool) {
		n = len(pool)
	}
	out := make([]string, 0, n)
	used := map[int]struct{}{}
	for len(out) < n {
		i := rand.Intn(len(pool))
		if _, ok := used[i]; ok {
			continue
		}
		used[i] = struct{}{}
		out = append(out, pool[i])
	}
	return out
}

func randomSentence() string {
	return gofakeit.Sentence(10)
}

func randomParagraph() string {
	n := rand.Intn(3) + 2 // 2–4 sentences
	parts := make([]string, 0, n)
	for i := 0; i < n; i++ {
		parts = append(parts, randomSentence())
	}
	return strings.Join(parts, " ")
}
