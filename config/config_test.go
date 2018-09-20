package config

import (
	"testing"
	"time"
)

func TestFormatNewIndexName(t *testing.T) {
	dt := time.Date(2018, 9, 8, 12, 23, 59, 0, time.UTC)
	r := RolloverAlias{
		NewName: `logs-write-%Y-%m-%d-%H%M%s`,
	}

	exp := `logs-write-2018-09-08-122359`
	res := r.NewIndexName(dt)

	if res != exp {
		t.Fatalf(`'%s' != '%s'`, exp, res)
	}
}
