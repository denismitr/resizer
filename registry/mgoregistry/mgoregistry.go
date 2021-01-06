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

type Config struct {
	DB               string
	ImagesCollection string
}

type MongoRegistry struct {
	client *mongo.Client
	db     *mongo.Database
	images *mongo.Collection
	slices *mongo.Collection
}

func New(client *mongo.Client, cfg Config) *MongoRegistry {
	r := MongoRegistry{
		client: client,
		db:     client.Database(cfg.DB),
	}

	r.images = r.db.Collection(cfg.ImagesCollection)
	r.slices = r.db.Collection("slices")

	return &r
}

func (r *MongoRegistry) CreateImageWithOriginalSlice(
	ctx context.Context,
	image *media.Image,
	slice *media.Slice,
) (imageID media.ID, sliceID media.ID, err error) {
	newImageID := primitive.NewObjectID()
	newSliceID := primitive.NewObjectID()
	txErr := r.transaction(ctx, 3*time.Second, func(sessCtx mongo.SessionContext) error {
		ir := mapImageToMongoRecord(image, newImageID)
		if err := r.createImage(sessCtx, ir); err != nil {
			return err
		}

		slice.ImageID = media.ID(ir.ID.Hex())

		sr := mapSliceToMongoRecord(slice, newSliceID)

		if err := r.createSlice(sessCtx, sr); err != nil {
			return err
		}

		return nil
	})

	if txErr != nil {
		return "", "", errors.Wrap(txErr, "could not create image and slice in one tx")
	}

	return media.ID(newImageID.Hex()), media.ID(newSliceID.Hex()), nil
}

func (r *MongoRegistry) GetImageByID(ctx context.Context, ID media.ID) (*media.Image, error) {
	var img *media.Image
	err := r.transaction(ctx, 2 * time.Second, func(sessCtx mongo.SessionContext) error {
		ir, err := r.getImageByID(sessCtx, ID)
		if err != nil {
			return err
		}

		sr, err := r.getOriginalSliceByImageID(sessCtx, ir.ID)
		if err != nil {
			return errors.Wrapf(err, "could not find original slice for image ID [%s]", ir.ID.Hex())
		}

		img = mapMongoRecordToImage(ir)
		img.OriginalSlice = mapMongoRecordToSlice(sr)

		return nil
	})

	if err != nil {
		return nil, errors.Wrapf(registry.ErrTxFailed, "mongo db closure failed, %v", err)
	}

	return img, nil
}

func (r *MongoRegistry) GetSliceByImageIDAndFilename(
	ctx context.Context,
	imageID media.ID,
	filename string,
) (*media.Slice, error) {
	var slice *media.Slice

	txErr := r.transaction(ctx, 2 * time.Second, func(sessCtx mongo.SessionContext) error {
		ID, err := primitive.ObjectIDFromHex(imageID.String())
		if err != nil {
			return errors.Wrapf(
				registry.ErrBadRegistryRequest,
				"invalid image ID [%s]: %v",
				imageID.String(), err.Error())
		}

		sr, err := r.getSliceByImageIDAndFilename(sessCtx, ID, filename);
		if err != nil {
			if errors.Is(err, registry.ErrSliceNotFound) || errors.Is(err, registry.ErrEntityNotFound) {
				return err
			}

			return errors.Wrapf(
				registry.ErrInternalRegistryError,
				"could not fetch slice for image #[%s]: %s",
				imageID.String(), err.Error())
		}

		slice = mapMongoRecordToSlice(sr)

		return nil
	})

	if txErr != nil {
		return nil, txErr
	}

	return slice, nil
}

func (r *MongoRegistry) GetImageAndExactMatchSliceIfExists(
	ctx context.Context,
	ID media.ID,
	filename string,
) (*media.Image, *media.Slice, error) {
	var img *media.Image
	var slice *media.Slice

	err := r.transaction(ctx, 2 * time.Second, func(sessCtx mongo.SessionContext) error {
		ir, err := r.getImageByID(sessCtx, ID)
		if err != nil {
			return err
		}

		osr, err := r.getOriginalSliceByImageID(sessCtx, ir.ID)
		if err != nil {
			return err
		}

		img = mapMongoRecordToImage(ir)
		img.OriginalSlice = mapMongoRecordToSlice(osr)

		if sr, err := r.getSliceByImageIDAndFilename(sessCtx, ir.ID, filename); err != nil {
			if ! errors.Is(err, registry.ErrSliceNotFound) {
				return err
			}
		} else {
			slice = mapMongoRecordToSlice(sr)
		}

		return nil
	})

	if err != nil {
		return nil, nil, errors.Wrapf(registry.ErrTxFailed, "mongo db tx failed; %v", err)
	}

	return img, slice, nil
}

func (r *MongoRegistry) CreateSlice(ctx context.Context, slice *media.Slice) (media.ID, error) {
	newID := primitive.NewObjectID()
	err := r.transaction(ctx, 3*time.Second, func(sessCtx mongo.SessionContext) error {
		sr := mapSliceToMongoRecord(slice, newID)

		if err := r.createSlice(sessCtx, sr); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return "", err
	}

	return media.ID(newID.Hex()), nil
}

func (r *MongoRegistry) CreateImage(ctx context.Context, img *media.Image) (media.ID, error) {
	newID := primitive.NewObjectID()
	err := r.transaction(ctx, 3 * time.Second, func(sessCtx mongo.SessionContext) error {

		ir := mapImageToMongoRecord(img, newID)

		if err := r.createImage(sessCtx, ir); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		if errors.Is(err, registry.ErrRegistryWriteFailed) { // fixme: more simple error structure
			return "", err
		}

		return "", errors.Wrapf(registry.ErrTxFailed, "create image tx failed: %v", err)
	}

	return media.ID(newID.Hex()), nil
}

func (r *MongoRegistry) transaction(ctx context.Context, commitTime time.Duration, f func(sessCtx mongo.SessionContext) error) error {
	wc := writeconcern.New(writeconcern.WMajority())
	rc := readconcern.Snapshot()

	txnOpts := options.Transaction().
		SetWriteConcern(wc).
		SetReadConcern(rc).
		SetMaxCommitTime(&commitTime)

	sess, err := r.client.StartSession()
	if err != nil {
		return errors.Wrapf(registry.ErrCouldNotOpenTx, "mongo db session failed %v", err)
	}

	defer sess.EndSession(ctx)

	_, txErr := sess.WithTransaction(ctx, func(sessCtx mongo.SessionContext) (interface{}, error) {
		if err := f(sessCtx); err != nil {
			return nil, err
		}

		return nil, nil
	}, txnOpts)

	if txErr != nil {
		return errors.Wrapf(registry.ErrTxFailed, "mongo db closure failed, %v", txErr)
	}

	return nil
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

func (r *MongoRegistry) createSlice(ctx mongo.SessionContext, sr *sliceRecord) error {
	if _, err := r.getSliceByImageIDAndFilename(ctx, sr.ImageID, sr.Filename); err == registry.ErrEntityNotFound {
		return errors.Wrapf(
			registry.ErrEntityAlreadyExists,
			"slice with image ID #[%s] and filename %s already exist",
				sr.ImageID.Hex(), sr.Filename)
	}

	result, err := r.slices.InsertOne(ctx, sr)
	if err != nil || result == nil {
		return errors.Wrapf(registry.ErrRegistryWriteFailed, "could not insert slice into MongoDB collection %v", err)
	}

	return nil
}

func (r *MongoRegistry) getSliceByImageIDAndFilename(
	ctx mongo.SessionContext,
	imageID primitive.ObjectID,
	filename string,
) (*sliceRecord, error) {
	var record sliceRecord
	if err := r.slices.FindOne(ctx, bson.M{"imageId": imageID, "filename": filename}).Decode(&record); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.Wrapf(
				registry.ErrEntityNotFound,
				"slice with image ID #[%s] and filename %s not found",
				imageID.Hex(), filename)
		}

		return nil, errors.Wrapf(
			registry.ErrRegistryReadFailed,
			"mongodb could not get slice with image ID [%s] and filename %s",
			imageID.String(), filename)
	}

	return &record, nil
}

func (r *MongoRegistry) getOriginalSliceByImageID(ctx mongo.SessionContext, imageID primitive.ObjectID) (*sliceRecord, error) {
	var record sliceRecord
	if err := r.slices.FindOne(ctx, bson.M{"imageId": imageID, "isOriginal": true}).Decode(&record); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, registry.ErrSliceNotFound
		}

		return nil, errors.Wrapf(
			registry.ErrRegistryReadFailed,
			"mongodb could not get slice with image ID [%s]",
			imageID.String())
	}

	return &record, nil
}
