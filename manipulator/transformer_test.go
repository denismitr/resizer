package manipulator

import (
	"bytes"
	"fmt"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"io"
	"os"
	"testing"
)

type testCloser func()

func openImageFile(path string) (io.Reader, testCloser) {
	f, err := os.Open(path)
	if err != nil {
		panic(err)
	}

	return f, func() { _ = f.Close() }
}

func createImageFile(path string) (io.Writer, testCloser) {
	f, err := os.OpenFile(path, os.O_CREATE | os.O_RDWR, 0655)
	if err != nil {
		panic(err)
	}

	return f, func() { _ = f.Close() }
}

func assertReaderEqualsFileContents(t *testing.T, path string, content io.Reader) {
	t.Helper()

	const chunkSize = 4096

	f, closer := openImageFile(path)
	defer closer()

	var i int
	for {
		buf1 := make([]byte, chunkSize)
		_, err1 := f.Read(buf1)

		buf2 := make([]byte, chunkSize)
		_, err2 := content.Read(buf2)

		if err1 != nil || err2 != nil {
			if err1 == io.EOF && err2 == io.EOF {
				t.Log("ok. reader contents equals file contents")
				return
			} else if err1 == io.EOF || err2 == io.EOF {
				t.Errorf("file content and passed reader have different length")
			} else {
				t.Errorf("%v - %v", err1, err2)
			}
		}

		if !bytes.Equal(buf1, buf2) {
			assert.Equal(t, buf1, buf2, fmt.Sprintf("iteration #%d", i))
			return
		}

		i++
	}
}

func TestManipulator_Transform_Flip(t *testing.T) {
	m := New(&Config{})

	t.Run("imageTransformer can flip vertically and reduce quality to 75, transforming from png to jpeg", func(t *testing.T) {
		transformation := &Transformation{
			Extension: JPEG,
			Quality:   75,
			Flip: Flip{
				Vertical: true,
				Horizontal: false,
			},
		}

		source, closeReader := openImageFile("./test_images/fishing.png")
		defer closeReader()

		dst := new(bytes.Buffer)

		r, err := m.Transform(source, dst, transformation)
		if err != nil {
			t.Fatal(err)
		}

		assert.NotNil(t, r)
		assertReaderEqualsFileContents(t, "./test_images/fishing_fv_q75.jpg", dst)
	})

	t.Run("imageTransformer can flip image horizontally and reduce quality to 90, transforming from png to jpeg", func(t *testing.T) {
		transformation := &Transformation{
			Extension: JPEG,
			Quality:   90,
			Flip: Flip{
				Vertical: false,
				Horizontal: true,
			},
		}

		source, closeReader := openImageFile("./test_images/fishing.png")
		defer closeReader()

		dst := new(bytes.Buffer)
		//dst, closer := createImageFile("./test_images/fishing_fh_q90.jpg")
		//defer closer()

		r, err := m.Transform(source, dst, transformation)
		if err != nil {
			t.Fatal(err)
		}

		assert.NotNil(t, r)
		//assert.FileExists(t, "./test_images/fishing_fh_q90.jpg")
		assertReaderEqualsFileContents(t, "./test_images/fishing_fh_q90.jpg", dst)
	})
}

func TestManipulator_Transform_Resize(t *testing.T) {
	m := New(&Config{})

	t.Run("imageTransformer can scale proportionally and preserve quality at 100, transforming from png to jpeg", func(t *testing.T) {
		transformation := &Transformation{
			Extension: JPEG,
			Quality:   100,
			Resize: Resize{
				Scale: 25,
			},
		}

		source, closeReader := openImageFile("./test_images/fishing.png")
		defer closeReader()

		dst := new(bytes.Buffer)

		r, err := m.Transform(source, dst, transformation)
		if err != nil {
			t.Fatal(err)
		}

		assert.NotNil(t, r)
		assertReaderEqualsFileContents(t, "./test_images/fishing_p25_q100.jpg", dst)
	})

	t.Run("imageTransformer can scale proportionally 60% and reduce quality to 50, transforming from jpeg to png", func(t *testing.T) {
		transformation := &Transformation{
			Extension: PNG,
			Quality:   50,
			Resize: Resize{
				Scale: 60,
			},
		}

		source, closeReader := openImageFile("./test_images/fishing_fh_q90.jpg")
		defer closeReader()

		dst := new(bytes.Buffer)
		//dst, closer := createImageFile("./test_images/fishing_p60_q50.png")
		//defer closer()

		r, err := m.Transform(source, dst, transformation)
		if err != nil {
			t.Fatal(err)
		}

		assert.NotNil(t, r)
		//assert.FileExists(t, "./test_images/fishing_p60_q50.png")
		assertReaderEqualsFileContents(t, "./test_images/fishing_p60_q50.png", dst)
	})

	t.Run("imageTransformer can scale by Height preserving side proportions", func(t *testing.T) {
		transformation := &Transformation{
			Extension: PNG,
			Quality:   50,
			Resize: Resize{
				Height: 400,
			},
		}

		source, closeReader := openImageFile("./test_images/fishing_fh_q90.jpg")
		defer closeReader()

		dst := new(bytes.Buffer)
		//dst, closer := createImageFile("./test_images/fishing_h400_q50.png")
		//defer closer()

		r, err := m.Transform(source, dst, transformation)
		if err != nil {
			t.Fatal(err)
		}

		assert.NotNil(t, r)
		//assert.FileExists(t, "./test_images/fishing_h400_q50.png")
		assertReaderEqualsFileContents(t, "./test_images/fishing_h400_q50.png", dst)
	})

	t.Run("imageTransformer can scale by width preserving side proportions", func(t *testing.T) {
		transformation := &Transformation{
			Extension: PNG,
			Quality:   55,
			Resize: Resize{
				Width: 450,
			},
		}

		source, closeReader := openImageFile("./test_images/fishing_fh_q90.jpg")
		defer closeReader()

		dst := new(bytes.Buffer)
		//dst, closer := createImageFile("./test_images/fishing_w450_q55.png")
		//defer closer()

		r, err := m.Transform(source, dst, transformation)
		if err != nil {
			t.Fatal(err)
		}

		assert.NotNil(t, r)
		//assert.FileExists(t, "./test_images/fishing_w450_q55.png")
		assertReaderEqualsFileContents(t, "./test_images/fishing_w450_q55.png", dst)
	})

	t.Run("imageTransformer will crop as needed when both height and width provided", func(t *testing.T) {
		transformation := &Transformation{
			Extension: PNG,
			Quality:   60,
			Resize: Resize{
				Width: 80,
				Height: 80,
			},
		}

		source, closeReader := openImageFile("./test_images/fishing_fh_q90.jpg")
		defer closeReader()

		//dst := new(bytes.Buffer)
		dst, closer := createImageFile("./test_images/fishing_w80_h80_q60.png")
		defer closer()

		r, err := m.Transform(source, dst, transformation)
		if err != nil {
			t.Fatal(err)
		}

		assert.NotNil(t, r)
		assert.FileExists(t, "./test_images/fishing_w80_h80_q60.png")
		//assertReaderEqualsFileContents(t, "./test_images/fishing_w80_h80_q60.png", dst)
	})

	t.Run("imageTransformer will return error if height is greater than original size", func(t *testing.T) {
		transformation := &Transformation{
			Extension: PNG,
			Quality:   55,
			Resize: Resize{
				Height: 3000,
			},
		}

		source, closeReader := openImageFile("./test_images/fishing_fh_q90.jpg")
		defer closeReader()

		dst := new(bytes.Buffer)

		r, err := m.Transform(source, dst, transformation)

		assert.Nil(t, r)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrBadTransformationRequest))
	})

	t.Run("imageTransformer will return error if width is greater than original size", func(t *testing.T) {
		transformation := &Transformation{
			Extension: PNG,
			Quality:   55,
			Resize: Resize{
				Width: 3000,
			},
		}

		source, closeReader := openImageFile("./test_images/fishing_fh_q90.jpg")
		defer closeReader()

		dst := new(bytes.Buffer)

		r, err := m.Transform(source, dst, transformation);
		assert.Error(t, err)
		assert.Nil(t, r)
		assert.True(t, errors.Is(err, ErrBadTransformationRequest))
	})
}

func TestManipulator_Crop(t *testing.T) {
	m := New(&Config{})

	t.Run("imageTransformer can crop only from left by given percent", func(t *testing.T) {
		transformation := &Transformation{
			Extension: JPEG,
			Quality:   75,
			Resize: Resize{
				Crop: Crop{
					Left: Percent(30),
				},
			},
		}

		source, closeReader := openImageFile("./test_images/tools.jpg")
		defer closeReader()

		dst := new(bytes.Buffer)
		//dst, closer := createImageFile("./test_images/tools_cl30.jpg")
		//defer closer()

		r, err := m.Transform(source, dst, transformation)
		if err != nil {
			t.Fatal(err)
		}

		assert.NotNil(t, r)
		//assert.FileExists(t, "./test_images/tools_cl30.jpg")
		assertReaderEqualsFileContents(t, "./test_images/tools_cl30.jpg", dst)
	})

	t.Run("imageTransformer can crop only from top by given percent", func(t *testing.T) {
		transformation := &Transformation{
			Extension: JPEG,
			Quality:   75,
			Resize: Resize{
				Crop: Crop{
					Top: Percent(30),
				},
			},
		}

		source, closeReader := openImageFile("./test_images/tools.jpg")
		defer closeReader()

		dst := new(bytes.Buffer)
		//dst, closer := createImageFile("./test_images/tools_ct30.jpg")
		//defer closer()

		r, err := m.Transform(source, dst, transformation)
		if err != nil {
			t.Fatal(err)
		}

		assert.NotNil(t, r)
		//assert.FileExists(t, "./test_images/tools_ct30.jpg")
		assertReaderEqualsFileContents(t, "./test_images/tools_ct30.jpg", dst)
	})

	t.Run("imageTransformer can crop only from right by given percent", func(t *testing.T) {
		transformation := &Transformation{
			Extension: JPEG,
			Quality:   75,
			Resize: Resize{
				Crop: Crop{
					Right: Percent(40),
				},
			},
		}

		source, closeReader := openImageFile("./test_images/tools.jpg")
		defer closeReader()

		dst := new(bytes.Buffer)
		//dst, closer := createImageFile("./test_images/tools_cr40.jpg")
		//defer closer()

		r, err := m.Transform(source, dst, transformation)
		if err != nil {
			t.Fatal(err)
		}

		assert.NotNil(t, r)
		//assert.FileExists(t, "./test_images/tools_cr40.jpg")
		assertReaderEqualsFileContents(t, "./test_images/tools_cr40.jpg", dst)
	})

	t.Run("imageTransformer can crop from all sides by equal percent", func(t *testing.T) {
		transformation := &Transformation{
			Extension: JPEG,
			Quality:   75,
			Resize: Resize{
				Crop: Crop{
					Left: Percent(20),
					Right: Percent(20),
					Top: Percent(20),
					Bottom: Percent(20),
				},
			},
		}

		source, closeReader := openImageFile("./test_images/tools.jpg")
		defer closeReader()

		dst := new(bytes.Buffer)
		//dst, closer := createImageFile("./test_images/tools_c20.jpg")
		//defer closer()

		r, err := m.Transform(source, dst, transformation)
		if err != nil {
			t.Fatal(err)
		}

		assert.NotNil(t, r)
		//assert.FileExists(t, "./test_images/tools_c20.jpg")
		assertReaderEqualsFileContents(t, "./test_images/tools_c20.jpg", dst)
	})
}

func TestCalculateDimensionAsProportion(t *testing.T) {
	// fixme
	var result int
	result = calculateDimensionAsProportion(500, 25)
	assert.Equal(t, 125, result)

	result = calculateDimensionAsProportion(100, 19)
	assert.Equal(t, 19, result)
}
