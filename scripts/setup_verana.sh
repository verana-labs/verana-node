#!/bin/bash

set -e

# Function to log messages
log() {
    echo "$(date '+%Y-%m-%d %H:%M:%S') - $1"
}

# Variables
CHAIN_ID="vna-local-1"
MONIKER="testdev"
BINARY="veranad"
HOME_DIR="$HOME/.verana"
GENESIS_JSON_PATH="$HOME_DIR/config/genesis.json"
APP_TOML_PATH="$HOME_DIR/config/app.toml"
CONFIG_TOML_PATH="$HOME_DIR/config/config.toml"
VALIDATOR_NAME="cooluser"
VALIDATOR_AMOUNT="1000000000000000000000uvna"

log "Starting Verana blockchain setup..."

# Initialize the chain
log "Initializing the chain..."
$BINARY init $MONIKER --chain-id $CHAIN_ID --default-denom uvna
if [ $? -ne 0 ]; then
    log "Error: Failed to initialize the chain."
    exit 1
fi

# Add a validator key
log "Adding validator key..."
$BINARY keys add $VALIDATOR_NAME --keyring-backend test
if [ $? -ne 0 ]; then
    log "Error: Failed to add validator key."
    exit 1
fi

# Add genesis account
log "Adding genesis account..."
$BINARY add-genesis-account $VALIDATOR_NAME $VALIDATOR_AMOUNT --keyring-backend test
if [ $? -ne 0 ]; then
    log "Error: Failed to add genesis account."
    exit 1
fi

# Two more keys, so the council has the three corporations genesis requires
for key in council_member1 council_member2; do
    $BINARY keys add $key --keyring-backend test
    $BINARY add-genesis-account $key 1000000000000uvna --keyring-backend test
done

# Seat the council: one corporation per key, the validator key's runs this node; short windows (overridable)
log "Seating the council of corporations in genesis..."
COUNCIL_FILE="$HOME_DIR/config/council.json"
PUBKEY=$($BINARY comet show-validator)
cat > "$COUNCIL_FILE" <<EOF
{"corporations":[
 {"did":"did:example:council-1","language":"en","members":["$VALIDATOR_NAME"],"funds":"10000000uvna","validator":{"moniker":"$MONIKER","pubkey":$PUBKEY}},
 {"did":"did:example:council-2","language":"en","members":["council_member1"],"funds":"10000000uvna"},
 {"did":"did:example:council-3","language":"en","members":["council_member2"],"funds":"10000000uvna"}]}
EOF
$BINARY genesis add-council "$COUNCIL_FILE" \
    --voting-period "${COUNCIL_VOTING_PERIOD:-30s}" \
    --min-execution-period "${COUNCIL_MIN_EXECUTION_PERIOD:-10s}" \
    --unbonding-time "${COUNCIL_UNBONDING_TIME:-60s}" \
    --keyring-backend test
if [ $? -ne 0 ]; then
    log "Error: Failed to seat the council."
    exit 1
fi

# Update minimum-gas-prices in app.toml
log "Updating minimum gas prices..."
sed -i '' 's/^minimum-gas-prices = ""/minimum-gas-prices = "0.25uvna"/' "$APP_TOML_PATH"
if [ $? -ne 0 ]; then
    log "Error: Failed to update minimum gas prices in app.toml."
    exit 1
fi

# Validate genesis file
log "Validating genesis file..."
$BINARY validate-genesis
if [ $? -ne 0 ]; then
    log "Error: Genesis file validation failed."
    exit 1
fi

# Update app.toml
log "Updating app.toml..."
sed -i '' 's/enable = false/enable = true/' "$APP_TOML_PATH"
sed -i '' 's/swagger = false/swagger = true/' "$APP_TOML_PATH"
sed -i '' 's/enabled-unsafe-cors = false/enabled-unsafe-cors = true/' "$APP_TOML_PATH"
if [ $? -ne 0 ]; then
    log "Error: Failed to update app.toml."
    exit 1
fi

# Update config.toml CORS settings
log "Updating CORS settings in config.toml..."
sed -i '' 's/cors_allowed_origins = \[\]/cors_allowed_origins = \["*"\]/' "$CONFIG_TOML_PATH"
if [ $? -ne 0 ]; then
    log "Error: Failed to update CORS settings in config.toml."
    exit 1
fi

# Start the chain
log "Starting the Verana blockchain..."
$BINARY start

log "Setup complete. If you encounter any issues, please check the logs above."