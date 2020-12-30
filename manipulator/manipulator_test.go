package manipulator

import (
	"bytes"
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
			t.Error("byte slices are not equal")
			return
		}
	}
}

func TestManipulator_Transform_Flip(t *testing.T) {
	t.Run("it can flip vertically and reduce quality to 75, transforming from png to jpeg", func(t *testing.T) {
		transformation := Transformation{
			Format: JPEG,
			Quality: 75,
			Flip: Flip{
				Vertical: true,
				Horizontal: false,
			},
		}

		source, closeReader := openImageFile("./test_images/fishing.png")
		defer closeReader()

		dst := new(bytes.Buffer)

		if err := (&Manipulator{}).Transform(source, dst, transformation); err != nil {
			t.Fatal(err)
		}

		assertReaderEqualsFileContents(t, "./test_images/fishing_fv_q75.jpg", dst)
	})

	t.Run("it can flip image horizontally and reduce quality to 90, transforming from png to jpeg", func(t *testing.T) {
		transformation := Transformation{
			Format: JPEG,
			Quality: 90,
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

		if err := (&Manipulator{}).Transform(source, dst, transformation); err != nil {
			t.Fatal(err)
		}

		//assert.FileExists(t, "./test_images/fishing_fh_q90.jpg")
		assertReaderEqualsFileContents(t, "./test_images/fishing_fh_q90.jpg", dst)
	})
}

