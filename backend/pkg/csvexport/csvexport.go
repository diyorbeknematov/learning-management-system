// Package csvexport writes a slice of structs as CSV. The columns are the
// fields that have a `csv:"name"` tag, in the order of the struct.
package csvexport

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// Write writes the header and one line per item of rows, which must be a slice
// of structs (or of pointers to structs). An empty slice gives only the
// header.
func Write(w io.Writer, rows any) error {
	value := reflect.ValueOf(rows)

	if value.Kind() != reflect.Slice {
		return errors.New("csvexport: rows must be a slice")
	}

	itemType := value.Type().Elem()
	if itemType.Kind() == reflect.Pointer {
		itemType = itemType.Elem()
	}

	if itemType.Kind() != reflect.Struct {
		return errors.New("csvexport: rows must be a slice of structs")
	}

	var header []string

	var fields []int

	for i := 0; i < itemType.NumField(); i++ {
		name := itemType.Field(i).Tag.Get("csv")
		if name == "" || name == "-" {
			continue
		}

		header = append(header, name)
		fields = append(fields, i)
	}

	if len(header) == 0 {
		return errors.New("csvexport: the struct has no csv tags")
	}

	writer := csv.NewWriter(w)

	if err := writer.Write(header); err != nil {
		return err
	}

	for i := 0; i < value.Len(); i++ {
		item := value.Index(i)
		if item.Kind() == reflect.Pointer {
			item = item.Elem()
		}

		record := make([]string, len(fields))
		for j, field := range fields {
			record[j] = format(item.Field(field))
		}

		if err := writer.Write(record); err != nil {
			return err
		}
	}

	writer.Flush()

	return writer.Error()
}

var timeType = reflect.TypeOf(time.Time{})

// format turns one field into text. A nil pointer is an empty cell.
func format(value reflect.Value) string {
	for value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return ""
		}

		value = value.Elem()
	}

	if value.Type() == timeType {
		return value.Interface().(time.Time).UTC().Format(time.RFC3339)
	}

	if stringer, ok := value.Interface().(fmt.Stringer); ok {
		return stringer.String()
	}

	switch value.Kind() {
	case reflect.String:
		return safe(value.String())
	case reflect.Bool:
		return strconv.FormatBool(value.Bool())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(value.Int(), 10)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.FormatUint(value.Uint(), 10)
	case reflect.Float32, reflect.Float64:
		return strconv.FormatFloat(value.Float(), 'f', -1, 64)
	}

	return safe(fmt.Sprint(value.Interface()))
}

// safe keeps a text from being run as a formula when the file is opened in a
// spreadsheet: a cell that starts with = + - @ (or a tab or a line break) gets
// a quote in front.
func safe(text string) string {
	if text != "" && strings.ContainsRune("=+-@\t\r", rune(text[0])) {
		return "'" + text
	}

	return text
}
