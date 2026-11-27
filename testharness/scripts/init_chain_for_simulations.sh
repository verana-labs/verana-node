#!/bin/bash
# scripts/init_chain_for_simulations.sh
# 
# This script initializes a fresh Verana blockchain chain optimized for TD yield simulations.
# It sets up:
# - A council of 3 corporations (one per key: cooluser, council_member1, council_member2), 30s ballot, no min execution delay
# - All necessary configuration for running simulations

set -e

# Function to log messages
log() {
    echo "$(date '+%Y-%m-%d %H:%M:%S') - $1"
}

# Use sed -i.bak for portable in-place editing (works on both macOS and Linux)
sedi() { sed -i.bak "$@" && rm -f "${@: -1}.bak"; }

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
SEED_PHRASE_COOLUSER="pink glory help gown abstract eight nice crazy forward ketchup skill cheese"
SEED_PHRASE_COUNCIL_MEMBER1="answer effort virtual mother muffin stadium buddy air enact luggage burger slim couch brass sight merge legend aspect clown boss august wish unknown wasp"
SEED_PHRASE_COUNCIL_MEMBER2="kitchen enough rough enforce bamboo easily exchange bullet error phone please razor claim quarter bounce stereo execute path pool series invite lucky warfare next"

# Default ports for primary validator
P2P_PORT="26656"
RPC_PORT="26657"
API_PORT="1317"
GRPC_PORT="9090"
GRPC_WEB_PORT="9091"

log "Starting Primary Validator setup for TD Yield Simulations..."

# Initialize the chain
log "Initializing the chain..."
$BINARY init $MONIKER --chain-id $CHAIN_ID

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

# Replace all occurrences of "stake" with "uvna" in genesis.json
log "Replacing 'stake' with 'uvna' in genesis.json..."
sedi 's/stake/uvna/g' "$GENESIS_JSON_PATH"

# Council members (vote only; cooluser is the seated validator)
log "Adding council member keys and genesis accounts..."
echo "$SEED_PHRASE_COUNCIL_MEMBER1" | $BINARY keys add council_member1 --recover --keyring-backend test
echo "$SEED_PHRASE_COUNCIL_MEMBER2" | $BINARY keys add council_member2 --recover --keyring-backend test
$BINARY add-genesis-account council_member1 1000000000000uvna --keyring-backend test
$BINARY add-genesis-account council_member2 1000000000000uvna --keyring-backend test

# Seat the council: one corporation per key, cooluser's runs this node; short windows for simulations
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
    --voting-period 30s \
    --min-execution-period 0s \
    --unbonding-time 60s \
    --keyring-backend test

if [ $? -ne 0 ]; then
    log "Error: Failed to seat the council."
    exit 1
fi

# Set minimum-gas-prices to 0uvna (allows 0 fees for accurate balance tracking)
log "Setting minimum gas prices to 0uvna (allows 0 fees for simulations)..."
sedi 's/^minimum-gas-prices = .*/minimum-gas-prices = "0uvna"/' "$APP_TOML_PATH"

# Configure ports in app.toml
sedi "s/:1317/:$API_PORT/" "$APP_TOML_PATH"
sedi "s/:9090/:$GRPC_PORT/" "$APP_TOML_PATH"
sedi "s/:9091/:$GRPC_WEB_PORT/" "$APP_TOML_PATH"

# Configure ports in config.toml
sedi "s/:26656/:$P2P_PORT/" "$CONFIG_TOML_PATH"
sedi "s/:26657/:$RPC_PORT/" "$CONFIG_TOML_PATH"

# Enable API and CORS
log "Updating API and CORS settings..."
sedi 's/enable = false/enable = true/' "$APP_TOML_PATH"
sedi 's/swagger = false/swagger = true/' "$APP_TOML_PATH"
sedi 's/enabled-unsafe-cors = false/enabled-unsafe-cors = true/' "$APP_TOML_PATH"
sedi 's/cors_allowed_origins = \[\]/cors_allowed_origins = \["*"\]/' "$CONFIG_TOML_PATH"

# Initialize YieldIntermediatePool module account with 1 uvna to prevent invariant violations
# This addresses the issue where empty module accounts break invariants when receiving funds
# See: https://github.com/cosmos/cosmos-sdk/issues/25315
log "Initializing YieldIntermediatePool module account with 1 uvna..."

# Hardcoded YieldIntermediatePool module account address (derived from module name)
YIELD_POOL_ADDR="verana1wjnrmvjlgxvs098cnu3jaczzjjm4csmqep067h"

if command -v python3 &> /dev/null; then
    python3 << PYEOF
import json
genesis_path = "$GENESIS_JSON_PATH"
yield_addr = "verana1wjnrmvjlgxvs098cnu3jaczzjjm4csmqep067h"

with open(genesis_path, "r") as f:
    genesis = json.load(f)

app_state = genesis.setdefault("app_state", {})
bank = app_state.setdefault("bank", {})
balances = bank.setdefault("balances", [])

# Find existing balance entry for the yield pool address
entry = next((b for b in balances if b.get("address") == yield_addr), None)
balance_added = False

if entry is None:
    # Create a new balance entry with 1 uvna
    balances.append({
        "address": yield_addr,
        "coins": [{"denom": "uvna", "amount": "1"}],
    })
    balance_added = True
else:
    # Update existing entry: increment or add uvna
    coins = entry.setdefault("coins", [])
    uvna = next((c for c in coins if c.get("denom") == "uvna"), None)
    if uvna is None:
        coins.append({"denom": "uvna", "amount": "1"})
        balance_added = True
    else:
        old_amount = int(uvna.get("amount", "0"))
        if old_amount == 0:
            uvna["amount"] = "1"
            balance_added = True
        else:
            uvna["amount"] = str(old_amount + 1)
            balance_added = True

# Update supply to match the added balance
if balance_added:
    supply = bank.setdefault("supply", [])
    uvna_supply = next((s for s in supply if s.get("denom") == "uvna"), None)
    if uvna_supply:
        # Increment existing supply by 1
        current_supply = int(uvna_supply.get("amount", "0"))
        uvna_supply["amount"] = str(current_supply + 1)
    else:
        # Add new supply entry
        supply.append({"denom": "uvna", "amount": "1"})

with open(genesis_path, "w") as f:
    json.dump(genesis, f, indent=2)

print(f"Ensured 1 uvna exists for YieldIntermediatePool account: {yield_addr}")
PYEOF
else
    log "Warning: python3 not found; cannot auto-initialize YieldIntermediatePool."
    log "Please add 1uvna to $YIELD_POOL_ADDR in $GENESIS_JSON_PATH manually."
fi

# Validate genesis file
log "Validating genesis file..."
$BINARY validate-genesis

# Save the genesis file for other validators
cp $GENESIS_JSON_PATH $HOME/genesis.json
log "Genesis file saved to $HOME/genesis.json"

# Get validator node ID
NODE_ID=$($BINARY tendermint show-node-id)
log "Primary Validator Node ID: $NODE_ID"

echo "Node ID: $NODE_ID" > $HOME/primary_validator_info.txt
echo "P2P Address: localhost:$P2P_PORT" >> $HOME/primary_validator_info.txt

# Print simulation configuration summary
log "=========================================="
log "Simulation Configuration Summary"
log "=========================================="
log "Council: corporations of cooluser, council_member1, council_member2 (2 of 3 to pass)"
log "Ballot: 30s, min execution: 0s"
log ""
log "Next steps:"
log "  1. Set FEES=0 environment variable for accurate balance tracking:"
log "     export FEES=0uvna"
log "  2. Start the chain: veranad start"
log "  3. Run proposal setup: go run cmd/main.go 20"
log "  4. Run simulations: go run cmd/main.go 21 (or 22)"
log ""
log "Note: FEES=0uvna is required to accurately track balances after reclaiming"
log "      (otherwise 750000uvna per transaction makes tracking difficult)"
log "=========================================="

# Start the chain
log "Starting the Primary Validator..."
$BINARY start

log "Primary Validator setup complete. If you encounter any issues, please check the logs above."

