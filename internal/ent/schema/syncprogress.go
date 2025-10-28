package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"entgo.io/ent/schema/mixin"
)

// SyncProgress holds the schema definition for the SyncProgress entity.
type SyncProgress struct {
	ent.Schema
}

func (SyncProgress) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.Time{},
	}
}

// Fields of the SyncProgress.
func (SyncProgress) Fields() []ent.Field {
	return []ent.Field{
		field.String("exchange"),
		field.String("account"),
		field.Int64("timestamp"),
	}
}

// Edges of the SyncProgress.
func (SyncProgress) Edges() []ent.Edge {
	return nil
}

// Indexes of the Event.
func (SyncProgress) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("exchange", "account").Unique(),
	}
}
