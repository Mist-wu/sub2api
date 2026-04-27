// Package schema defines Ent ORM schemas.
package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// UserImageGeneration stores free user-side image generation history.
type UserImageGeneration struct {
	ent.Schema
}

// Annotations returns schema annotations.
func (UserImageGeneration) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "user_image_generations"},
	}
}

// Fields defines all fields for user image generations.
func (UserImageGeneration) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("user_id"),
		field.String("prompt").
			SchemaType(map[string]string{dialect.Postgres: "text"}).
			NotEmpty(),
		field.String("revised_prompt").
			SchemaType(map[string]string{dialect.Postgres: "text"}).
			Optional().
			Nillable(),
		field.String("model").
			MaxLen(100).
			Default("gpt-image-2"),
		field.String("mime_type").
			MaxLen(100).
			Default("image/png"),
		field.Bytes("image_data").
			NotEmpty().
			SchemaType(map[string]string{dialect.Postgres: "bytea"}),
		field.String("image_sha256").
			MaxLen(64).
			NotEmpty(),
		field.Time("created_at").
			Default(time.Now).
			Immutable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
	}
}

// Edges defines relations.
func (UserImageGeneration) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("image_generations").
			Field("user_id").
			Required().
			Unique().
			Annotations(entsql.OnDelete(entsql.Cascade)),
	}
}

// Indexes defines query indexes.
func (UserImageGeneration) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id"),
		index.Fields("created_at"),
		index.Fields("user_id", "created_at"),
		index.Fields("image_sha256"),
	}
}
