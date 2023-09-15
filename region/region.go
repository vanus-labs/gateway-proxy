package region

import (
	"context"
	"fmt"

	"github.com/vanus-labs/gateway-proxy/db"
	"github.com/vanus-labs/gateway-proxy/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	regionCollName = "regions"
)

var (
	regionColl *mongo.Collection
	regions    = make([]models.RegionInfo, 0)
)

func Init(ctx context.Context, name models.Region) error {
	if regionColl == nil {
		regionColl = db.NewCollection(regionCollName)
	}
	query := bson.M{
		"name": name,
	}
	opt := &options.FindOptions{
		Sort: bson.M{"is_default": -1},
	}
	cursor, err := regionColl.Find(ctx, query, opt)
	if err != nil {
		return err
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()

	for cursor.Next(ctx) {
		region := &models.RegionInfo{}
		if err = cursor.Decode(region); err != nil {
			return err
		}
		if err = region.Validate(); err != nil {
			return err
		}
		regions = append(regions, *region)
	}
	if len(regions) == 0 {
		return fmt.Errorf("%s region info is empty", name)
	}
	if !regions[0].IsDefault {
		return fmt.Errorf("default cluster of %s does not exist", name)
	}
	return nil
}

func GetAllRegionInfo(ctx context.Context) []models.RegionInfo {
	return regions
}
