package db

import (
	"context"
	"fmt"
	"sync"
	"time"

	log "k8s.io/klog/v2"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
)

var (
	cfg  Config
	cli  *mongo.Client
	once sync.Once
)

type Config struct {
	Database string `yaml:"database"`
	Address  string `yaml:"address"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

func Init(ctx context.Context, c Config) (*mongo.Client, error) {
	var err error
	once.Do(func() {
		cfg = c
		start := time.Now()
		uri := fmt.Sprintf("mongodb+srv://%s:%s@%s/?retryWrites=true&w=majority", c.Username, c.Password, c.Address)
		clientOptions := options.Client().
			ApplyURI(uri).
			SetServerAPIOptions(options.ServerAPI(options.ServerAPIVersion1))
		ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		cli, err = mongo.Connect(ctx, clientOptions)
		if err != nil {
			return
		}
		if err = cli.Ping(ctx, readpref.Primary()); err != nil {
			return
		}
		log.Infof("spent %d ms success to connect to MongoDB, address: %s\n", time.Since(start).Milliseconds(), cfg.Address)
	})

	if cli != nil {
		return cli, nil
	}
	return cli, nil
}

func GetDatabaseName() string {
	return cfg.Database
}

func NewCollection(name string) *mongo.Collection {
	return cli.Database(cfg.Database).Collection(name)
}

func NewSession() (mongo.Session, error) {
	return cli.StartSession()
}

func TransactionOptions() *options.TransactionOptions {
	return options.Transaction().SetWriteConcern(
		writeconcern.New(writeconcern.WMajority()),
	)
}
