package models

import (
	"time"

	"github.com/gocql/gocql"
	"github.com/kristoiv/gocqltable"
	"github.com/kristoiv/gocqltable/recipes"
)

// ConversationEntity represents a conversation record in the database.
type ConversationEntity struct {
	ID        gocql.UUID `cql:"id"`
	OwnerId   gocql.UUID `cql:"owner_id"`
	Name      string     `cql:"name"`
	Type      int        `cql:"type"`
	CreatedAt time.Time  `cql:"created_at"`
}

// ConversationRepository provides CRUD operations for the conversations table.
var ConversationRepository = struct {
	recipes.CRUD
}{
	recipes.CRUD{
		TableInterface: gocqltable.NewKeyspace("ks_chat").NewTable(
			"conversations",
			[]string{"id"},
			nil,
			ConversationEntity{},
		),
	},
}
