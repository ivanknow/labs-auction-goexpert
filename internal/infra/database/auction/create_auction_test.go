package auction

import (
	"context"
	"os"
	"testing"
	"time"

	"fullcycle-auction_go/internal/entity/auction_entity"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func TestCreateAuction_CompletesAfterConfiguredDuration(t *testing.T) {
	t.Setenv("AUCTION_DURATION", "200ms")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db := connectToMongoForTest(t)
	repo := NewAuctionRepository(db)
	collection := db.Collection("auctions")
	if err := collection.Drop(ctx); err != nil && err.Error() != "mongo: no documents in result" {
		// ignore cleanup failures and continue
	}
	defer func() {
		_ = collection.Drop(ctx)
	}()

	auction, err := auction_entity.CreateAuction(
		"Test Product",
		"Electronics",
		"A valid description for the auction",
		auction_entity.New,
	)
	if err != nil {
		t.Fatalf("expected auction to be created: %v", err)
	}

	if internalErr := repo.CreateAuction(ctx, auction); internalErr != nil {
		t.Fatalf("expected auction creation to succeed: %v", internalErr)
	}

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		foundAuction, internalErr := repo.FindAuctionById(ctx, auction.Id)
		if internalErr != nil {
			t.Fatalf("expected auction to be retrievable: %v", internalErr)
		}

		if foundAuction.Status == auction_entity.Completed {
			return
		}

		time.Sleep(50 * time.Millisecond)
	}

	t.Fatalf("expected auction status to become %v after the configured duration", auction_entity.Completed)
}

func connectToMongoForTest(t *testing.T) *mongo.Database {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	mongoURL := os.Getenv("MONGODB_URL")
	if mongoURL == "" {
		mongoURL = "mongodb://admin:admin@127.0.0.1:27017/auctions?authSource=admin"
	}

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURL))
	if err != nil {
		t.Skipf("mongo not available: %v", err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		t.Skipf("mongo not available: %v", err)
	}

	dbName := os.Getenv("MONGODB_DB")
	if dbName == "" {
		dbName = "auctions"
	}

	return client.Database(dbName)
}
