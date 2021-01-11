package mgoregistry

import (
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"resizer/media"
	"resizer/registry"
)

func (r *MongoRegistry) getImages(ctx mongo.SessionContext, imageFilter media.ImageFilter) ([]imageRecord, int64, error) {
	var records []imageRecord

	filter := bson.M{}
	if imageFilter.Bucket != "" {
		filter["bucket"] = imageFilter.Bucket // fixme: rename to namespace
	}

	opts := options.Find()
	opts.SetSkip(int64(imageFilter.Offset()))
	opts.SetLimit(int64(imageFilter.Limit()))

	cursor, err := r.images.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, errors.Wrapf(registry.ErrRegistryReadFailed, "mongodb could not find images by filter %v", filter)
	}

	if err := cursor.All(ctx, &records); err != nil {
		return nil, 0, errors.Wrapf(registry.ErrRegistryReadFailed, "mongodb could not decode images", filter)
	}

	total, err := r.images.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, errors.Wrapf(registry.ErrRegistryReadFailed, "mongodb could not count images", filter)
	}

	return records, total, nil
}

func (r *MongoRegistry) getImageByID(ctx mongo.SessionContext, ID media.ID) (*imageRecord, error) {
	objectID, err := primitive.ObjectIDFromHex(ID.String())
	if err != nil {
		return nil, registry.ErrInvalidID
	}

	var record imageRecord
	if err := r.images.FindOne(ctx, bson.M{"_id": objectID}).Decode(&record); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, registry.ErrEntityNotFound
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
	if err := r.slices.FindOne(ctx, bson.M{
		"imageId": imageID,
		"filename": filename,
		"status": media.Ready,
	}).Decode(&record); err != nil {
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
	if err := r.slices.FindOne(ctx, bson.M{
		"imageId": imageID,
		"isOriginal": true,
		"status": media.Ready,
	}).Decode(&record); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, registry.ErrEntityNotFound
		}

		return nil, errors.Wrapf(
			registry.ErrRegistryReadFailed,
			"mongodb could not get slice with image ID [%s]: %v",
			imageID.String(), err)
	}

	return &record, nil
}
