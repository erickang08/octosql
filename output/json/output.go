package json

import (
	"encoding/json"
	"io"

	"github.com/pkg/errors"

	"github.com/cube2222/octosql/execution"
	"github.com/cube2222/octosql/output"
)

type Output struct {
	w                  io.Writer
	enc                *json.Encoder
	firstRecordWritten bool
}

func NewOutput(w io.Writer) output.Output {
	return &Output{
		w:                  w,
		enc:                json.NewEncoder(w),
		firstRecordWritten: false,
	}
}

func (o *Output) WriteRecord(record *execution.Record) error {
	if !o.firstRecordWritten {
		o.firstRecordWritten = true
		n, err := o.w.Write([]byte{'['})
		if err != nil {
			return errors.Wrap(err, "couldn't write leading square bracket")
		}
		if n != 1 {
			return errors.Errorf("couldn't write leading square bracket")
		}
	} else {
		n, err := o.w.Write([]byte{','})
		if err != nil {
			return errors.Wrap(err, "couldn't write separating comma")
		}
		if n != 1 {
			return errors.Errorf("couldn't write separating comma")
		}
	}
	kvs := make(map[string]interface{})
	for _, field := range record.Fields() {
		kvs[field.Name.String()] = record.Value(field.Name).ToRawValue()
	}
	err := o.enc.Encode(kvs)
	if err != nil {
		return errors.Wrap(err, "couldn't encode record as json")
	}

	return nil
}

func (o *Output) Close() error {
	n, err := o.w.Write([]byte{']'})
	if err != nil {
		return errors.Wrap(err, "couldn't write trailing square bracket")
	}
	if n != 1 {
		return errors.Errorf("couldn't write trailing square bracket")
	}

	return nil
}
