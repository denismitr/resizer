# Resizing image proxy
### On the fly resizer and proxy for your images written in pure GO

#### Requires MongoDB replicaSet and S3 compatible storage

## WIP

### Structure
Project currently consists of two applications: backoffice and resizing proxy. Backoffice is an API that is
used for uploading images, configuring their settings, removing them. UI for the backend API is TODO. Proxy
has only one endpoint `{host}:{port}/v1/images/{imageId}/{transformations}.{extension}`. In `transformations` you can specify
desired transformations separated by underscore `_` and extension is just one of allowed extension (png, jpg - for now). 
Example:
```go
localhost:3333/v1/images/5ffacaab456cb200a1ad1dd0/cl20_h300_q80.png
```
that will crop image from the left by 20%, limit height by 300px and reduce quality from the original to 80% 
and return image encoded as PNG

### Available transformation segments
* h\d{1,5} - height (if width not provided - proportion will be preserved)
* w\d{1,5} - width (if height not provided - proportion will be preserved)
* w\d{1,5}_h\d{1,5} - width and height (proportion will be changed)
* s\d{1,3} - scale down by percent (if scaleUp is disabled - cannot be larger than 100)
* fh - flip horizontally
* fv - flip vertically
* bw - black & white (TODO)
* o\d{1,2} - opacity in percent (TODO)
* r90 - rotate 90 degrees
* r180 - rotate 90 degrees
* r270 - rotate 90 degrees
* cl\d{1,2} - crop left by percent
* cr\d{1,2} - crop right by percent
* ct\d{1,2} - crop top by percent
* cb\d{1,2} - crop bottom by percent
* c\d{1,2} - crop all sides by percent

### Caching
All resized images are stored in a storage (S3 only for now). It happens asyncronously,
after resized image is returned to the client. Additional Redis cache is TODO. On the next
request with similar transformation requirements resized image will be fetched from the storage.

### Discrete steps
To prevent DDoS attacks you can specify discrete steps for all transformations. So, for example,
a client will not be able to actually get images different by one pixel in height. Instead a closest possible
result will be returned. E.g. if you set `DISCRETE_SIZE_STEP` env var to 50, and your original image has height of
230px than proxy will return only heights: 230px, 180px, 130px, 80px and 30px, fetching nearest possible height to the
requested one.

### Backoffice
Has 3 routes at the moment
* GET {host}:{port}/api/v1/images/{id} get an image by ID
* POST {host}:{port}/api/v1/images - create an image (upload)
* GET {host}:{port}/api/v1/images - get images with pagination and filter
* DELETE {host}:{port}/api/v1/images/{id} - delete an image and all sliced copies from the registry and storage

### TODO
* Swagger for backoffice
* Support webp
* UI for backoffice
* More transformations (rotation, opacity)
* Transformation on image create
* Watermark on image create
* Auto crop
* Redis caching
* MySQL as alternative DB
* Azure and Google cloud as alternative storage


