`schema.sql` is created by:

```bash
export PGPASSWORD='testpwd1'
pg_dump \
  --schema-only \
  --no-owner \
  --no-privileges \
  --no-tablespaces \
  --disable-triggers \
  --host=localhost \
  --port=5432 \
  --username=postgres \
  <<database>> > schema.sql
```
