// Package integration holds end-to-end tests that run the real services
// (resource engine, results, links, documents, payments) against a private,
// fully migrated Postgres schema. They're behind the "integration" build tag:
//
//	make test-integration          # or:
//	TEST_DATABASE_URL=postgres://… go test -tags integration ./internal/integration/
package integration
