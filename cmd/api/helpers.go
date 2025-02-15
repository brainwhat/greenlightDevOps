package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/julienschmidt/httprouter"
)

func (app *application) readIDParams(r *http.Request) (int64, error) {

	params := httprouter.ParamsFromContext(r.Context())

	id, err := strconv.ParseInt(params.ByName("id"), 10, 64)
	if err != nil || id < 1 {
		return 0, errors.New("invalid id parameter")
	}

	return id, nil
}

type envelope map[string]any

func (app *application) writeJSON(w http.ResponseWriter, status int, data envelope, headers http.Header) error {

	js, err := json.MarshalIndent(data, "", "\t")
	if err != nil {
		return err
	}

	js = append(js, '\n')

	for key, value := range headers {
		w.Header()[key] = value
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(js)

	return nil
}

// We are working around every error that json.Decode() can return
func (app *application) readJSON(w http.ResponseWriter, r *http.Request, dst any) error {

	// Protect against DOS attacks
	maxBytes := 1_048_576
	r.Body = http.MaxBytesReader(w, r.Body, int64(maxBytes))

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	err := dec.Decode(dst)
	if err != nil {
		// This one is for convenience
		var syntaxError *json.SyntaxError
		var unmarshalTypeError *json.UnmarshalTypeError
		var invalidUnmarshalError *json.InvalidUnmarshalError
		var maxBytesError *http.MaxBytesError

		switch {
		// For some reason Decode() can return either json.SyntaxError or io.ErrUnexpectedEOF
		// When dealing with JSON syntax errors. So we check for both

		// errors.As() checks is the err matches the TYPE, so we can access the fields
		case errors.As(err, &syntaxError):
			return fmt.Errorf("body contains JSON syntax error at character %d", syntaxError.Offset)

		case errors.Is(err, io.ErrUnexpectedEOF):
			return errors.New("body contains JSON syntax error")

		// Check if json type doesn't match the destination type. Show the dst field if possible
		case errors.As(err, &unmarshalTypeError):
			if unmarshalTypeError.Field != "" {
				return fmt.Errorf("body contains invalid JSON type for field %q", unmarshalTypeError.Field)
			}
			return fmt.Errorf("body contains invalid JSON type at character %d", unmarshalTypeError.Offset)

		// Check if req body is empty
		case errors.Is(err, io.EOF):
			return errors.New("Request body cannot be empty")

		case strings.HasPrefix(err.Error(), "json: unknown field "):
			fieldName := strings.TrimPrefix(err.Error(), "json: unknown field ")
			return fmt.Errorf("body contains uknown key %s", fieldName)

		case errors.As(err, &maxBytesError):
			return fmt.Errorf("body must be under %d bytes", maxBytesError.Limit)

		// We panic here because this error is caused by faulty code only
		case errors.As(err, &invalidUnmarshalError):
			panic(err)

		default:
			return err
		}
	}

	// Check if there is anything else after the json object
	err = dec.Decode(&struct{}{})
	if !errors.Is(err, io.EOF) {
		return errors.New("body must contain only one json value")
	}

	return nil
}
