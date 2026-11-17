package gomypal

import (
	"fmt"
	"io"
)

type request struct {
	Op          string
	Method      string
	Path        string
	UID         string
	RequireUID  bool
	SandboxOnly bool
	Body        any
	File        *filePart
}

type filePart struct {
	FieldName string
	FileName  string
	Content   io.Reader
}

type required struct {
	name  string
	empty bool
}

func checkRequired(op string, fields ...required) error {
	for _, field := range fields {
		if field.empty {
			return fmt.Errorf("gomypal: %s: %s: %w", op, field.name, ErrMissingField)
		}
	}

	return nil
}
