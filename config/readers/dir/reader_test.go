package dir

import (
	log "github.com/cantara/bragi/sbragi"
	"os"
	"strings"
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
	if len(files) != 3 {
		log.Error("wrong amount of files were found")
		t.Fatal("wrong amount of files were found")
	}
	if !files[0].Binary {
		t.Fatalf("File was not binary as expected")
	}
	if !(strings.HasPrefix(files[1].Mode, "06") && strings.HasSuffix(files[1].Mode, "0")) {
		t.Fatalf("File mode was not as expected: %s != %s", files[1].Mode, "0640")
	}
	if !(strings.HasPrefix(files[2].Mode, "17") && strings.HasSuffix(files[2].Mode, "0")) {
		t.Fatalf("File mode was not as expected: %s != %s", files[2].Mode, "1750")
	}
	if files[1].Binary {
		t.Fatalf("File was binary as not expected")
	}
	if files[2].Binary {
		t.Fatalf("File was binary as not expected")
	}
}
