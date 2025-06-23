# Sentra Layer AVS

## AVS consensus

For each AVS, all of its operators have to interface through a middleware layer to relay/receive data on chain (receive tasks and return their result). In this layer, we have a configurable value `quorum` which is a percentage value indicate the threshold of stake amount needed consensus. That is, for every task emitted, each of the AVS's operators should return a signed message containing result to the middleware layer, if the amount of stake delegated to the operators having submitted reaches the `quorum` then we confirm a result for that task, in other word, the AVS has reached consensus for that task. 

![alt text](consensus.png)

## Structure

### Main components

### Objects / Structs

1. `avs::service_manager::ServiceManagerStore`:
- Created for each cluster of AVS services.
- Stores necessary data for AVS task verification.
- Can be customized for each type of AVS service (e.g., verifying ETH -> Aptos bridging results).

```move
    struct ServiceManagerStore has key {
        tasks_state: SmartTable<vector<u8>, TaskState>,
        tasks_creator: SmartTable<u64, address>,
        task_count: u64,
    }

    struct TaskState has store, drop, copy {
        task_created_timestamp: u64,
        responded: bool,
        response: u128,
        data_request: String,
        respond_fee_token: Object<Metadata>,
        respond_fee_limit: u64
    }
```

2. `avs::service_manager::TaskCreatorBalanceStore`:
- Created for each cluster of AVS services.
- Keep track token balance of each task creator in avs
- When a task creator deposit some token to the contract, fa will be store in fee pool and the balance of that token will be store here. 

```move
    struct TaskCreatorBalanceStore has key, store {
        balances: SmartTable<Object<Metadata>, u64>,
    }
```

3. `avs::service_manager::FeePool`:
- Created for each cluster of AVS services.
- Stores FA for task when a task creator deposit, it will be distribute to operator that fulfill the task with correct data.

```move
    struct FeePool has key {
        token_store: Object<FungibleStore>
    }
```

4. `avs::stake_registry::StakeRegistryStore`:
- Created for each cluster of AVS services.
- Used to have an overview of stakes of operators
- Keep track total stake, minimum stake for each quorum 

```move
    struct  StakeRegistryStore has key {
        total_stake_history: SmartTable<u8, vector<StakeUpdate>>,
        strategy_params: SmartTable<u8, vector<StrategyParams>>,
        minimum_stake_for_quorum: SmartTable<u8, u128>,
        operator_stake_history: SmartTable<vector<u8>, SmartTable<u8, vector<StakeUpdate>>>,
    }

    struct StakeUpdate has copy, drop, store {
        update_timestamp : u64, 
        next_update_timestamp : u64,
        stake: u128,
    }

    struct StrategyParams has copy, drop, store {
        strategy: Object<Metadata>,
        multiplier: u128,
    }
```

5. `avs::registry_coordinator::RegistryCoordinatorStore`
- Created for each cluster of AVS services.
- Keep track numbers of quorum, each quorum params, each operator infos, operator bitmap and operator bitmap history

```move
    struct RegistryCoordinatorStore has key {
        quorum_count: u8,
        quorum_params: SmartTable<u8, OperatorSetParam>,
        operator_infos: SmartTable<address, OperatorInfo>,
        operator_bitmap: SmartTable<vector<u8>, u256>,
        operator_bitmap_history: SmartTable<vector<u8>, vector<QuorumBitmapUpdate>>,
    }

    struct OperatorInfo has copy, drop, store {
        operator_id: vector<u8>,
        operator_status: u8, // 0: NEVER_REGISTERED, 1: REGISTERED, 2: DEREGISTERED
    }

    struct QuorumBitmapUpdate has copy, drop, store {
        update_timestamp: u64,
        next_update_timestamp: u64, 
        quorum_bitmap: u256,
    }

    struct OperatorSetParam has copy, drop, store {
        max_operator_count: u32,
    }
```

6. `avs::index_registry::IndexRegistryStore`
- Created for each cluster of AVS services.
- To have an overview of the list of operators 

```move
    struct IndexRegistryStore has key {
        operator_index: SmartTable<u8, SmartTable<String, u32>>,
        update_history: SmartTable<u8, SmartTable<u32, vector<OperatorUpdate>>>,
        count_history: SmartTable<u8, vector<QuorumUpdate>>
    }

    struct OperatorUpdate has copy, store, drop {
        operator_id: String, 
        timestamp: u64
    }

    struct QuorumUpdate has copy, store, drop {
        operator_count: u32,
        timestamp: u64
    }
```

7. `avs::bls_apk_registry::BLSApkRegistryStore`
- Created for each cluster of AVS services.
- Stores necessary data for bls pubkey aggregation process.

```move
    struct BLSApkRegistryStore has key {
        operator_to_pk_hash: SmartTable<address, vector<u8>>,
        pk_hash_to_operator: SmartTable<vector<u8>, address>,
        operator_to_pk: SmartTable<address, PublicKeyWithPoP>,
        apk_history: SmartTable<u8, vector<ApkUpdate>>,
        current_apk: SmartTable<u8, vector<PublicKeyWithPoP>>
    }

    struct ApkUpdate has store, drop {
        aggregate_pubkeys: Option<AggrPublicKeysWithPoP>,
        update_timestamp: u64,
        next_update_timestamp: u64
    }
```

8. `avs::fee_pool::FeePool`
- Created for each cluster of AVS services.
- Stores token to distribute later.

```move
    struct FeePool has key {
        token_store: Object<FungibleStore>,
    }
```

9. 
### Modules

1. `avs::registry_coordinator`:
- Entry point to create quorums, register operator to the avs, deregister
- Keep track of `stake_registry`, `index_registry`, `bls_apk_registry`.

2. `avs::index_registry`:
- Keep track of operators set and it's index 
- Managed by `registry_coordinator`

3. `avs::stake_registry`:
- Keep track total stake for each quorum, minumum stake for each quorum, staking token params 
- Record operator stake update for each register, deregister operator event
- Managed by `registry_coordinator`

4. `avs::bls_apk_registry`:
- Record the current aggregate BLS pubkey for all Operators registered to each quorum
- Managed by `registry_coordinator`

5. `avs::bls_sig_checker`:
- Perform on chain BLS signature validation for the aggregate of a quorum's registered Operators.

6. `avs::fee_pool`:
- Store the whole contract's fungible asset 

6. `avs::service_manager`:
- Endpoint for user to create tasks and for operator to resolve tasks
- 

7. Helper modules

- `restaking::epoch`: calculate epoch from timestamp (neccessary for slashing)

- `restaking::math`: math functions like bytes32-u256 converters

- `restaking::merkle_tree`: verify merkle proof (neccessary for rewards claim verification)

- `restaking::slashing_accounting`: calculate the shares of operators before / after slashed

### Public interface (entry functions)

#### Register/deregister Operator

##### Module: `avs::registry_coordinator`

1. Register Operator

```move
    public entry fun registor_operator(
      operator: &signer,  
      quorum_numbers: vector<u8>, 
      signature: vector<u8>, 
      pubkey: vector<u8>, 
      pop: vector<u8>
    ) acquires RegistryCoordinatorStore
```

Called by operator to register to provide security for AVS.

`quorum_numbers`: quorum numbers operator register for

`signature`: used to verify signature share

`pubkey`: BLS public key of operator

`pop`: proof of possession of the public key above

2. Deregister Operator

```move
    public entry fun deregister_operator(
      operator: &signer, 
      quorumNumbers: vector<u8>
    ) acquires RegistryCoordinatorStore
```

Called by operator to deregistor from quorums

`quorumNumbers` : quorums to deregister

3. Create quorum

```move
      public entry fun create_quorum(
        owner: &signer,
        operator_set_params: OperatorSetParam { max_operator_count : u32}, 
        minumum_stake: u128, 
        strategies: vector<address>, 
        multipliers: vector<u128>
      ) acquires RegistryCoordinatorStore, RegistryCoordinatorConfigs
    
```

Create a quorum and record it configuration. Can only called by owner of the AVS.

`operator_set_params`: maximum operator could be created in a quorum
`minumum_stake`: minimum stake amount for a operator to join the quorum
`strategies`: FA address that the quorum allow
`multipliers`: used for calculate operator's weight in quorum (corresponding with strategies)

4. Update operators for quorum

Aggregator will call this function repeatedly to update the operators in quorum

```move
     public entry fun update_operators_for_quorum(
        aggregator: &signer,
        quorum_numbers: vector<u8>,
        opertors_per_quorum: vector<vector<address>>,
    ) acquires RegistryCoordinatorConfigs, RegistryCoordinatorStore
```

`quorum_numbers`: quorums that need update operator
`opertors_per_quorum`: operator adresses each quorum

5. Set operator set param

```move
    public entry fun set_operator_set_params(
      owner: &signer,
      quorum_number: u8, 
      operator_set_params: OperatorSetParam
    ) acquires RegistryCoordinatorStore
```

Updates an existing quorum's configuration. Can only called by owner of the AVS.

`quorum_number`: quorum to update configuration

`operator_set_params`: new operator set params

#### Create new tasks, respond tasks

##### Module: `avs::service_manager`

1. Create new task

```move
    public entry fun create_new_task(
        creator: &signer,
        token: Object<Metadata>,
        amount: u64,
        data_request: String,
        respond_fee_limit: u64
    ) acquires ServiceManagerConfigs, ServiceManagerStore, TaskCreatorBalanceStore
```

Called by anyone, mainly AVS consumer to have a new task created.

`token`: fee to create new tasks

`amount`: fee amount

`data_request`: the data requested inside the task 

`respond_fee_limit`: the fee limit for the task responder.

2. Respond to task

```move
    public entry fun respond_to_task(
        aggregator: &signer,
        task_id: u64,
        responses: vector<u128>,
        signer_pubkeys: vector<vector<u8>>,
        signer_sigs: vector<vector<u8>>,
    ) acquires ServiceManagerStore
```

Called by aggregator to resolve a task.
 
`task_id`: the id of the task to respond to

`responses`: the price that signed operators respond to aggregator

`signer_pubkeys`: pubkey of operator that ulfill the task, the hash of it will be used as operator id

`signer_sigs`: signature to prove that the price is signed by operator

### View functions

```move

public fun get_operator_id(operator: address): vector<u8> acquires RegistryCoordinatorStore

public fun get_operator_status(operator: address): u8 acquires RegistryCoordinatorStore

public fun get_operator_address(operator_id: vector<u8>): address acquires RegistryCoordinatorStore

public fun get_quorum_bitmap_by_timestamp(operator_id: vector<u8>, timestamp: u64): u256 acquires RegistryCoordinatorStore

public fun get_operator_bitmap_history_length(operator_id: vector<u8>): u64 acquires RegistryCoordinatorStore

public fun get_current_quorum_bitmap(operator_id: vector<u8>): u256 acquires RegistryCoordinatorStore

public fun quorum_count(): u8 acquires RegistryCoordinatorStore

public fun total_stake_at_timestamp_from_index(quorum_number: u8, timestamp: u64, index: u64): u128 acquires StakeRegistryStore

public fun get_stake_at_timestamp_and_index(quorum_number: u8, timestamp: u64, operator_id: vector<u8>, index: u64): u128 acquires StakeRegistryStore

public fun total_history_length(quorum_number: u8): u64 acquires StakeRegistryStore

public fun minimum_stake(quorum_number: u8): u128 acquires StakeRegistryStore

public fun strategy_params_length(quorum_number: u8): u64 acquires StakeRegistryStore

public fun strategy_by_index(quorum_number: u8, index: u64): Object<Metadata> acquires StakeRegistryStore

public fun count_history(quorum_number: u8): vector<QuorumUpdate> acquires IndexRegistryStore

public fun get_operator_id(operator: address): vector<u8> acquires BLSApkRegistryStore

public fun get_aggr_pk_hash_at_timestamp(quorum_number: u8, timestamp: u64, index: u64): vector<u8> acquires BLSApkRegistryStore

public fun get_operator_pk(operator: address): PublicKeyWithPoP acquires BLSApkRegistryStore

public fun validate_signature(operator_id: vector<u8>, signature: vector<u8>, msg: vector<u8>): bool acquires BLSApkRegistryStore

public fun check_signatures(quorum_numbers: vector<u8>, reference_timestamp: u64, msg_hashes: vector<vector<u8>>, signer_pubkeys: vector<vector<u8>>, signer_sigs: vector<vector<u8>>):(vector<u128>, vector<u128>)

public fun task_count(): u64  acquires ServiceManagerStore

public fun get_msg_hash(task_id: u64, response: u128): vector<u8> acquires ServiceManagerStore

public fun get_msg_hashes(task_id: u64, responses: vector<u128>, signer_pubkeys: vector<vector<u8>>): vector<vector<u8>> acquires ServiceManagerStore

public fun task_by_id(task_id: u64): TaskState acquires ServiceManagerStore

```