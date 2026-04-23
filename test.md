# Testing and Verification

## API Connectivity Verification

Run these steps in order to verify local configuration, OAuth, and Whoop API connectivity end-to-end:

```bash
# Build CLI
go build -o whooper .

# Configure credentials from developer.whoop.com
./whooper config set client-id <your-client-id>
./whooper config set client-secret <your-client-secret>

# Optional: confirm config values (secret is masked)
./whooper config

# Run smoke checks (local + API)
./whooper doctor

# Complete OAuth login flow in your browser
./whooper login

# Sync data from Whoop API
./whooper sync

# Verify data export works
./whooper export -e recoveries -f json
```

Expected outcome:

- `whooper doctor` reports all `[ok]` checks and prints `Doctor checks passed.`
- `whooper login` saves a token to `~/.whooper/token.json`
- `whooper sync` prints record counts per entity and ends with `Sync complete!`

## Non-API Smoke Mode

If you want to verify only local setup (without live API calls):

```bash
./whooper doctor --skip-api
```
