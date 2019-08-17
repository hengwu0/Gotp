package files

import "testing"

func TestFiles(t *testing.T) {
	filename := "/home/wuheng"
	if a, _ := FileType(filename); a != UNKNOWN {
		tmp := make([]byte, 10)
		tmp[1] = 100
		tmp[6] = 200
		t.Errorf("%x!\n", tmp)
	}
}
