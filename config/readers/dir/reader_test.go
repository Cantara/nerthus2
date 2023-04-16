package dir

import (
	log "github.com/cantara/bragi/sbragi"
	"os"
	"testing"
)

func TestReader(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		log.WithError(err).Error("while getting wd")
		t.Fatal(err)
	}
	files, err := ReadFilesFromDir(os.DirFS(wd), "test_data", "data")
	if err != nil {
		log.WithError(err).Error("while reading files")
		t.Fatal(err)
	}
	if len(files) != 2 {
		log.Error("wrong amount of files were found")
		t.Fatal("wrong amount of files were found")
	}
	if files[0].Mode != "0664" {
		t.Fatalf("File mode was not as expected: %s != %s", files[0].Mode, "0664")
	}
	if files[1].Mode != "1755" {
		t.Fatalf("File mode was not as expected: %s != %s", files[1].Mode, "1755")
	}
}
