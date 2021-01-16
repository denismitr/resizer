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
	SlicesCollection string
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
	r.slices = r.db.Collection(cfg.SlicesCollection)

	return &r
}

func (r *MongoRegistry) GenerateID() media.ID {
	return media.ID(primitive.NewObjectID().Hex())
}

func (r *MongoRegistry) Migrate(ctx context.Context) error {
	_, err := r.slices.Indexes().CreateOne(
		ctx,
		mongo.IndexModel{
			Keys: bson.M{
				"imageId": 1,
				"filename": 1,
			},
			Options: options.Index().SetUnique(true),
		},
	)

	if err != nil {
		return errors.Wrap(err, "could not create index on slices collection")
	}

	return nil
}

func (r *MongoRegistry) CreateImageWithOriginalSlice(
	ctx context.Context,
	image *media.Image,
	slice *media.Slice,
) (imageID media.ID, sliceID media.ID, err error) { // fixme: return only error
	txErr := r.transaction(ctx, 3*time.Second, func(sessCtx mongo.SessionContext) error {
		ir := mapImageToMongoRecord(image)
		if err := r.createImage(sessCtx, ir); err != nil {
			return err
		}

		sr := mapSliceToMongoRecord(slice)

		if err := r.createSlice(sessCtx, sr); err != nil {
			return err
		}

		return nil
	})

	if txErr != nil {
		return "", "", errors.Wrap(txErr, "could not create image and slice in one tx")
	}

	return image.ID, slice.ID, nil // fixme: return only nil
}

func (r *MongoRegistry) DepublishImage(ctx context.Context, ID media.ID) error {
	imageID, err := primitive.ObjectIDFromHex(ID.String())
	if err != nil {
		return registry.ErrInvalidID
	}

	return r.transaction(ctx, 2 * time.Second, func(sessCtx mongo.SessionContext) error {
		if err := r.depublishImage(sessCtx, imageID); err != nil {
			return err
		}

		return nil
	})
}

func (r *MongoRegistry) GetImageByID(ctx context.Context, ID media.ID, onlyPublished bool) (*media.Image, error) {
	var img *media.Image

	imageID, err := primitive.ObjectIDFromHex(ID.String())
	if err != nil {
		return nil, registry.ErrInvalidID
	}

	txErr := r.transaction(ctx, 2 * time.Second, func(sessCtx mongo.SessionContext) error {
		ir, err := r.getImageByID(sessCtx, imageID, onlyPublished)
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

	if txErr != nil {
		return nil, txErr
	}

	return img, nil
}

// GetImageWithSlicesByID - get image and all of it's slices including the original by image ID
func (r *MongoRegistry) GetImageWithSlicesByID(ctx context.Context, ID media.ID, onlyPublished bool) (*media.Image, error) {
	var img *media.Image
	imageID, err := primitive.ObjectIDFromHex(ID.String())
	if err != nil {
		return nil, registry.ErrInvalidID
	}

	txErr := r.transaction(ctx, 2 * time.Second, func(sessCtx mongo.SessionContext) error {
		ir, err := r.getImageByID(sessCtx, imageID, onlyPublished)
		if err != nil {
			return err
		}

		slicesRecords, err := r.getAllSlicesByImageID(sessCtx, ir.ID)
		if err != nil {
			return errors.Wrapf(err, "could not find original slice for image ID [%s]", ir.ID.Hex())
		}

		img = mapMongoRecordToImage(ir)
		for _, sr := range slicesRecords {
			img.Slices = append(img.Slices, *mapMongoRecordToSlice(&sr))
		}

		if img.Slices == nil {
			img.Slices = make(media.Slices, 0)
		}

		return nil
	})

	if txErr != nil {
		return nil, txErr
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
			if errors.Is(err, registry.ErrEntityNotFound) {
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

func (r *MongoRegistry) GetImages(ctx context.Context, imageFilter media.ImageFilter) (*media.ImageCollection, error) {
	collection := new(media.ImageCollection)

	txErr := r.transaction(ctx, 3 * time.Second, func(sessCtx mongo.SessionContext) error {
		records, total, err := r.getImages(sessCtx, imageFilter)
		if err != nil {
			return err
		}

		for _, r := range records {
			collection.Images = append(collection.Images, *mapMongoRecordToImage(&r))
		}

		if collection.Images == nil {
			collection.Images = make([]media.Image, 0)
		}

		collection.Meta.Total = uint(total)
		collection.Meta.PerPage = imageFilter.PerPage
		collection.Meta.Page = imageFilter.Page

		return nil
	})

	if txErr != nil {
		return nil, txErr
	}

	return collection, nil
}

func (r *MongoRegistry) GetImageAndExactMatchSliceIfExists(
	ctx context.Context,
	ID media.ID,
	filename string,
	onlyPublished bool,
) (*media.Image, *media.Slice, error) {
	var img *media.Image
	var slice *media.Slice

	imageID, err := primitive.ObjectIDFromHex(ID.String())
	if err != nil {
		return nil, nil, registry.ErrInvalidID
	}

	txErr := r.transaction(ctx, 2 * time.Second, func(sessCtx mongo.SessionContext) error {
		ir, err := r.getImageByID(sessCtx, imageID, onlyPublished)
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
			if ! errors.Is(err, registry.ErrEntityNotFound) {
				return err
			}
		} else {
			slice = mapMongoRecordToSlice(sr)
		}

		return nil
	})

	if txErr != nil {
		return nil, nil, txErr
	}

	return img, slice, nil
}

func (r *MongoRegistry) RemoveImageWithAllSlices(ctx context.Context, ID media.ID) error {
	return r.transaction(ctx, 3*time.Second, func(sessCtx mongo.SessionContext) error {
		imageID, err := primitive.ObjectIDFromHex(ID.String())
		if err != nil {
			return errors.Wrapf(
				registry.ErrBadRegistryRequest,
				"invalid image ID [%s]: %v",
				ID.String(), err.Error())
		}

		if err := r.removeAllSlicesByImageId(sessCtx, imageID); err != nil {
			return errors.Wrapf(err, "could not remove image with all slices")
		}

		if err := r.removeImage(sessCtx, imageID); err != nil {
			return errors.Wrapf(err, "could not remove image with all slices")
		}

		return nil
	})
}

func (r *MongoRegistry) CreateSlice(ctx context.Context, slice *media.Slice) (media.ID, error) {
	newID := primitive.NewObjectID()
	err := r.transaction(ctx, 3*time.Second, func(sessCtx mongo.SessionContext) error {
		sr := mapSliceToMongoRecord(slice)

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

		ir := mapImageToMongoRecord(img)

		if err := r.createImage(sessCtx, ir); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return "", nil
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
		return txErr
	}

	return nil
}

