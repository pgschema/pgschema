module github.com/pgschema/pgschema

go 1.24.0

toolchain go1.24.7

require (
	github.com/BurntSushi/toml v1.5.0
	github.com/fergusstrange/embedded-postgres v1.29.0
	github.com/google/go-cmp v0.7.0
	github.com/jackc/pgx/v5 v5.7.5
	github.com/joho/godotenv v1.5.1
	github.com/pganalyze/pg_query_go/v6 v6.1.0
	github.com/pgschema/pgschema/ir v0.0.0
	github.com/spf13/cobra v1.9.1
)

require (
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/lib/pq v1.10.9 // indirect
	github.com/spf13/pflag v1.0.6 // indirect
	github.com/stretchr/testify v1.10.0 // indirect
	github.com/xi2/xz v0.0.0-20171230120015-48954b6210f8 // indirect
	golang.org/x/crypto v0.37.0 // indirect
	golang.org/x/sync v0.17.0 // indirect
	golang.org/x/text v0.24.0 // indirect
	google.golang.org/protobuf v1.36.5 // indirect
)

replace github.com/pgschema/pgschema/ir => ./ir
