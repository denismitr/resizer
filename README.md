# Resizing image proxy
### On the fly resizer and proxy for your images written in pure GO

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

### TODO
