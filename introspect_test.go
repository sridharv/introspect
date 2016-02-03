package introspect_test

import (
	"fmt"
	"testing"

	"encoding/json"

	"github.com/sridharv/fail"
	"github.com/sridharv/introspect"
)

func TestParse(t *testing.T) {
	defer fail.Using(t.Fatal)

	builder, err := introspect.NewFileBuilder("testdata/test.go.src")
	fail.IfErr(err)

	f, err := builder.Build()
	fail.IfErr(err)

	j, err := json.MarshalIndent(f, "", "  ")
	fail.IfErr(err)
	fmt.Println(string(j))
}
