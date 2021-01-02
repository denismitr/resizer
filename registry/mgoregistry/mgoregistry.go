package mgoregistry

import (
	"context"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readconcern"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
	"resizer/media"
	"resizer/registry"
	"time"
)

type imageRecord struct {
	ID           primitive.ObjectID `bson:"_id"`
	Name         string             `bson:"name"`
	OriginalName string             `bson:"originalName"`
	OriginalSize int                `bson:"originalSize"`
	OriginalExt  string             `bson:"originalExt"`
	PublishAt    *time.Time         `bson:"publishedAt"`
	CreatedAt    time.Time          `bson:"createdAt"`
	UpdatedAt    time.Time          `bson:"updatedAt"`
	Bucket       string             `bson:"bucket"`
	Path         string             `bson:"path"`
	Url          string             `bson:"url"`
}

type Config struct {
	DB               string
	ImagesCollection string
}

type MongoRegistry struct {
	client *mongo.Client
	db     *mongo.Database
	images *mongo.Collection
}

func New(client *mongo.Client, cfg Config) *MongoRegistry {
	r := MongoRegistry{
		client: client,
		db:     client.Database(cfg.DB),
	}

	r.images = r.db.Collection(cfg.ImagesCollection)

	return &r
}

func (r *MongoRegistry) GetImageByID(ctx context.Context, ID media.ID) (*media.Image, error) {
	wc := writeconcern.New(writeconcern.WMajority())
	rc := readconcern.Snapshot()
	txnOpts := options.Transaction().SetWriteConcern(wc).SetReadConcern(rc)

	sess, err := r.client.StartSession()
	if err != nil {
		return nil, errors.Wrapf(registry.ErrCouldNotOpenTx, "mongo db session failed %v", err)
	}

	defer sess.EndSession(ctx)

	var img *media.Image
	_, txErr := sess.WithTransaction(ctx, func(sessCtx mongo.SessionContext) (interface{}, error) {
		ir, err := r.getImageByID(sessCtx, ID)
		if err != nil {
			return nil, err
		}

		img = mapMongoRecordToImage(ir)

		return ir, nil
	}, txnOpts)

	if txErr != nil {
		return nil, errors.Wrapf(registry.ErrTxFailed, "mongo db closure failed, %v", txErr)
	}

	return img, nil
}

func (r *MongoRegistry) CreateImage(ctx context.Context, img *media.Image) (media.ID, error) {
	wc := writeconcern.New(writeconcern.WMajority())
	rc := readconcern.Snapshot()
	txTtl := 3 * time.Second // fixme
	txnOpts := options.Transaction().
		SetWriteConcern(wc).
		SetReadConcern(rc).
		SetMaxCommitTime(&txTtl)

	sess, err := r.client.StartSession()
	if err != nil {
		return "", errors.Wrapf(registry.ErrCouldNotOpenTx, "mongo db session failed %v", err)
	}

	defer sess.EndSession(ctx)

	newID := primitive.NewObjectID()
	_, txErr := sess.WithTransaction(ctx, func(sessCtx mongo.SessionContext) (interface{}, error) {

		ir := mapImageToMongoRecord(img, newID)

		if err := r.createImage(sessCtx, ir); err != nil {
			return nil, err
		}

		return nil, nil
	}, txnOpts)

	if txErr != nil {
		if errors.Is(txErr, registry.ErrRegistryWriteFailed) {
			return "", txErr
		}

		return "", errors.Wrapf(registry.ErrTxFailed, "mongo db closure failed, %v", txErr)
	}

	return media.ID(newID.Hex()), nil
}

func (r *MongoRegistry) getImageByID(ctx mongo.SessionContext, ID media.ID) (*imageRecord, error) {
	objectID, err := primitive.ObjectIDFromHex(ID.String())
	if err != nil {
		panic(err) // fixme
	}

	var record imageRecord
	if err := r.images.FindOne(ctx, bson.M{"_id": objectID}).Decode(&record); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, registry.ErrImageNotFound
		}

		return nil, errors.Wrapf(registry.ErrRegistryReadFailed, "mongodb could not get image with id %s", ID.String())
	}

	return &record, nil
}

func (r *MongoRegistry) createImage(ctx mongo.SessionContext, ir *imageRecord) error {
	result, err := r.images.InsertOne(ctx, ir)
	if err != nil || result == nil {
		return errors.Wrapf(registry.ErrRegistryWriteFailed, "could not insert image into MongoDB collection %v", err)
	}

	return nil
}

func mapMongoRecordToImage(ir *imageRecord) *media.Image {
	return &media.Image{
		ID:           media.ID(ir.ID.Hex()),
		OriginalName: ir.OriginalName,
		Name:         ir.Name,
		OriginalExt:  ir.OriginalExt,
		Bucket:       ir.Bucket,
		Path:         ir.Path,
		Url:          ir.Url,
		CreatedAt:    ir.CreatedAt,
		UpdatedAt:    ir.UpdatedAt,
		PublishAt:    ir.PublishAt,
		OriginalSize: ir.OriginalSize,
	}
}

func mapImageToMongoRecord(img *media.Image, mongoID primitive.ObjectID) *imageRecord {
	if img.ID.None() && mongoID.IsZero() {
		panic("how can both media ID and mongo ID be empty")
	}

	ir := imageRecord{
		Name:         img.Name,
		OriginalName: img.OriginalName,
		OriginalSize: img.OriginalSize,
		OriginalExt:  img.OriginalExt,
		Bucket:       img.Bucket,
		Path:         img.Path,
		Url:          img.Url,
		CreatedAt:    img.CreatedAt,
		UpdatedAt:    img.UpdatedAt,
		PublishAt:    img.PublishAt,
	}

	if img.ID.None() {
		ir.ID = mongoID
	} else {
		if ID, err := primitive.ObjectIDFromHex(img.ID.String()); err != nil {
			panic("how can image ID be invalid")
		} else {
			ir.ID = ID
		}
	}

	return &ir
}
