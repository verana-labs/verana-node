#!/bin/bash
# scripts/init_local_chain.sh
#
# This script initializes a fresh local Verana blockchain for development and testing.
# It sets up:
# - A single validator node with the 'cooluser' account
# - A 3-seat council (cooluser + 2 voting members), 30s ballot, 10s min execution
# - API, gRPC, and CORS enabled
#
# Usage:
#   rm -rf ~/.verana && bash scripts/init_local_chain.sh
#   # Then in a new terminal: bash scripts/setup_group.sh

set -e

# Function to log messages
log() {
    echo "$(date '+%Y-%m-%d %H:%M:%S') - $1"
}

# Variables
CHAIN_ID="vna-testnet-1"
MONIKER="validator1"
BINARY="veranad"
HOME_DIR="$HOME/.verana"
GENESIS_JSON_PATH="$HOME_DIR/config/genesis.json"
APP_TOML_PATH="$HOME_DIR/config/app.toml"
CONFIG_TOML_PATH="$HOME_DIR/config/config.toml"
VALIDATOR_NAME="cooluser"
VALIDATOR_AMOUNT="1000000000000000000000uvna"
GENTX_AMOUNT="1000000uvna"
SEED_PHRASE_COOLUSER="pink glory help gown abstract eight nice crazy forward ketchup skill cheese"
SEED_PHRASE_COUNCIL_MEMBER1="answer effort virtual mother muffin stadium buddy air enact luggage burger slim couch brass sight merge legend aspect clown boss august wish unknown wasp"
SEED_PHRASE_COUNCIL_MEMBER2="kitchen enough rough enforce bamboo easily exchange bullet error phone please razor claim quarter bounce stereo execute path pool series invite lucky warfare next"

# Default ports
P2P_PORT="26656"
RPC_PORT="26657"
API_PORT="1317"
GRPC_PORT="9090"
GRPC_WEB_PORT="9091"

log "=========================================="
log "  Initializing Local Verana Chain"
log "=========================================="

# Initialize the chain with uvna as default denom
log "Initializing the chain..."
$BINARY init $MONIKER --chain-id $CHAIN_ID --default-denom uvna

if [ $? -ne 0 ]; then
    log "Error: Failed to initialize the chain."
    exit 1
fi

# Add a validator key
log "Adding validator key..."
echo "$SEED_PHRASE_COOLUSER" | $BINARY keys add $VALIDATOR_NAME --recover --keyring-backend test

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

# Council members (vote only; cooluser is the seated validator)
log "Adding council member keys and genesis accounts..."
echo "$SEED_PHRASE_COUNCIL_MEMBER1" | $BINARY keys add council_member1 --recover --keyring-backend test
echo "$SEED_PHRASE_COUNCIL_MEMBER2" | $BINARY keys add council_member2 --recover --keyring-backend test
$BINARY add-genesis-account council_member1 1000000000000uvna --keyring-backend test
$BINARY add-genesis-account council_member2 1000000000000uvna --keyring-backend test

# Replace remaining "stake" references BEFORE gentx
# Use Python for reliable cross-platform editing (macOS sed -i '' has issues with bash variables)
log "Replacing 'stake' with 'uvna' in genesis.json..."
python3 - <<'PYEOF'
import os

home = os.path.expanduser("~")
genesis_path = os.path.join(home, ".verana", "config", "genesis.json")

with open(genesis_path) as f:
    content = f.read()

with open(genesis_path, "w") as f:
    f.write(content.replace('"stake"', '"uvna"'))
PYEOF

# Seat the council: 3 weight-1 members, 2/3 policy, short windows for local testing
log "Seating the council in genesis..."
$BINARY genesis add-council cooluser,council_member1,council_member2 \
    --voting-period 30s \
    --min-execution-period 10s \
    --unbonding-time 60s \
    --keyring-backend test

# Create gentx (bond_denom=uvna is now set in genesis)
log "Creating genesis transaction..."
$BINARY gentx $VALIDATOR_NAME $GENTX_AMOUNT \
    --chain-id $CHAIN_ID \
    --moniker $MONIKER \
    --commission-rate "0.10" \
    --commission-max-rate "0.20" \
    --commission-max-change-rate "0.01" \
    --min-self-delegation "1" \
    --keyring-backend test

if [ $? -ne 0 ]; then
    log "Error: Failed to create genesis transaction."
    exit 1
fi

# Collect genesis transactions
log "Collecting genesis transactions..."
$BINARY collect-gentxs

# Configure app.toml and config.toml using Python
log "Configuring app.toml and config.toml..."
python3 - <<PYEOF
import os

home = os.path.expanduser("~")
app_toml = os.path.join(home, ".verana", "config", "app.toml")
config_toml = os.path.join(home, ".verana", "config", "config.toml")

with open(app_toml) as f:
    c = f.read()
c = c.replace('minimum-gas-prices = ""', 'minimum-gas-prices = "0uvna"')
c = c.replace('enable = false', 'enable = true')
c = c.replace('swagger = false', 'swagger = true')
c = c.replace('enabled-unsafe-cors = false', 'enabled-unsafe-cors = true')
c = c.replace(':1317', ':${API_PORT}')
c = c.replace(':9090', ':${GRPC_PORT}')
c = c.replace(':9091', ':${GRPC_WEB_PORT}')
with open(app_toml, 'w') as f:
    f.write(c)

with open(config_toml) as f:
    c = f.read()
c = c.replace('cors_allowed_origins = []', 'cors_allowed_origins = ["*"]')
c = c.replace(':26656', ':${P2P_PORT}')
c = c.replace(':26657', ':${RPC_PORT}')
with open(config_toml, 'w') as f:
    f.write(c)
PYEOF

# Validate genesis file
log "Validating genesis file..."
$BINARY validate-genesis

# Save the genesis file
cp $GENESIS_JSON_PATH $HOME/genesis.json
log "Genesis file saved to $HOME/genesis.json"

# Get validator node ID
NODE_ID=$($BINARY tendermint show-node-id)
log "Validator Node ID: $NODE_ID"

log ""
log "=========================================="
log "  ✅ Local Chain Initialized"
log "=========================================="
log ""
log "  Chain ID:  $CHAIN_ID"
log "  Validator: $VALIDATOR_NAME"
log "  RPC:       http://localhost:$RPC_PORT"
log "  API:       http://localhost:$API_PORT"
log "  gRPC:      localhost:$GRPC_PORT"
log ""
log "  Next steps:"
log "    1. Start the chain: veranad start"
log "    2. In a new terminal: bash scripts/setup_group.sh"
log ""
log "=========================================="
