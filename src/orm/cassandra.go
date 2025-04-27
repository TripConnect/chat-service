package orm

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/gocql/gocql"
)

func getCassandraSession(keyspace string) (*gocql.Session, error) {
	cluster := gocql.NewCluster("127.0.0.1") // change to your Cassandra IP
	cluster.Keyspace = keyspace
	cluster.Consistency = gocql.Quorum
	cluster.Authenticator = gocql.PasswordAuthenticator{
		Username: "username",
		Password: "password",
	}

	session, err := cluster.CreateSession()
	return session, err
}

func getAll[T any]() ([]*T, error) {
	var t T

	typ := reflect.TypeOf(t)
	specs_field, specs_found := typ.FieldByName("Specs")

	if !specs_found {
		return nil, errors.New("specs field required")
	}

	keyspace := specs_field.Tag.Get("keyspace")
	table := specs_field.Tag.Get("table")

	cql := fmt.Sprintf(
		"SELECT * FROM %s.%s",
		keyspace, table)

	session, session_error := getCassandraSession(keyspace)

	if session_error == nil {
		return nil, errors.New("database cannot connected")
	}

	defer session.Close()

	rows := session.Query(cql).Iter()

	var result []*T

	for rows.Scan(&t) {
		result = append(result, &t)
	}

	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("error closing rows: %v", err)
	}

	return result, nil
}
