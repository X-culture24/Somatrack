package api

import (
	"encoding/json"

	"github.com/google/uuid"
)

// OptionalUUID unmarshals a JSON value as a UUID but treats "" the same as
// null/absent instead of erroring. A plain *uuid.UUID rejects "" with
// "invalid UUID length: 0" — but an HTML <select> with nothing chosen, and
// an unresolved Postman collection variable, both commonly submit an empty
// string rather than omitting the field or sending null. Every
// client-optional UUID field in this package (class/stream/academic-year
// links, transport route, etc.) should use this instead of *uuid.UUID.
type OptionalUUID struct {
	UUID  uuid.UUID
	Valid bool
}

func (o *OptionalUUID) UnmarshalJSON(data []byte) error {
	var raw *string
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if raw == nil || *raw == "" {
		o.UUID = uuid.Nil
		o.Valid = false
		return nil
	}
	id, err := uuid.Parse(*raw)
	if err != nil {
		return err
	}
	o.UUID = id
	o.Valid = true
	return nil
}

func (o OptionalUUID) MarshalJSON() ([]byte, error) {
	if !o.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(o.UUID)
}

// Ptr returns nil when not set, else a pointer to the UUID — the shape pgx
// wants for a nullable uuid column.
func (o OptionalUUID) Ptr() *uuid.UUID {
	if !o.Valid {
		return nil
	}
	return &o.UUID
}

func OptionalUUIDFromPtr(id *uuid.UUID) OptionalUUID {
	if id == nil {
		return OptionalUUID{}
	}
	return OptionalUUID{UUID: *id, Valid: true}
}
