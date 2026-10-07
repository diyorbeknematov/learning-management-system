// Package authz builds the Casbin enforcer from the model and the policy that
// are compiled into the program.
package authz

import (
	"bytes"
	_ "embed"
	"encoding/csv"
	"fmt"
	"io"
	"strings"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
)

//go:embed model.conf
var modelText string

//go:embed policy.csv
var policyText string

// New builds the enforcer. The policy is a plain list in policy.csv (lines
// that start with # are comments); it is read once and never changed while the
// program runs.
func New() (*casbin.Enforcer, error) {
	m, err := model.NewModelFromString(modelText)
	if err != nil {
		return nil, fmt.Errorf("authz model: %w", err)
	}

	enforcer, err := casbin.NewEnforcer(m)
	if err != nil {
		return nil, fmt.Errorf("authz enforcer: %w", err)
	}

	reader := csv.NewReader(bytes.NewBufferString(policyText))
	reader.Comment = '#'
	reader.FieldsPerRecord = -1
	reader.TrimLeadingSpace = true

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}

		if err != nil {
			return nil, fmt.Errorf("authz policy: %w", err)
		}

		for i := range record {
			record[i] = strings.TrimSpace(record[i])
		}

		switch record[0] {
		case "p":
			_, err = enforcer.AddPolicy(record[1:])
		case "g":
			_, err = enforcer.AddGroupingPolicy(record[1:])
		default:
			err = fmt.Errorf("unknown line type %q", record[0])
		}

		if err != nil {
			return nil, fmt.Errorf("authz policy: %w", err)
		}
	}

	return enforcer, nil
}
