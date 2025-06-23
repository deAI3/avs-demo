module avs::registry_coordinator{
    use aptos_framework::event;
    use aptos_framework::fungible_asset::{
    Self, Metadata,
    };
    use aptos_framework::object::{Self, Object};
    use aptos_framework::account::{Self, SignerCapability};
    use aptos_framework::timestamp;
    use aptos_framework::primary_fungible_store;

    use avs::avs_manager;

    use restaking::staker_manager;

    use avs::service_manager_base;
    use avs::bls_apk_registry::{Self};
    use avs::stake_registry;
    use avs::index_registry;

    use restaking::operator_manager;

    use avs::math_utils;

    use aptos_std::smart_table::{Self, SmartTable};
    use aptos_std::smart_vector::{Self, SmartVector};
    use aptos_std::bls12381::{ Signature, PublicKeyWithPoP };
    use aptos_std::aptos_hash;
    use aptos_std::comparator;

    use std::string;
    use std::bcs;
    use std::vector;
    use std::signer;

    const REGISTRY_COORDINATOR_NAME: vector<u8> = b"REGISTRY_COORDINATOR_NAME";
    const REGISTRY_COORDINATOR_PREFIX: vector<u8> = b"REGISTRY_COORDINATOR_PREFIX";

    const REGISTER_MSG_HASH :vector<u8> = b"PubkeyRegistration";

    struct RegistryCoordinatorConfigs has key {
        signer_cap: SignerCapability,
    }
    struct RegistryCoordinatorStore has key {
        quorum_count: u8,
        quorum_params: SmartTable<u8, OperatorSetParam>,
        operator_infos: SmartTable<address, OperatorInfo>,
        operator_addresses: SmartTable<vector<u8>, address>,
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

    public entry fun initialize() {
        if (is_initialized()) {
            return
        };

        // derive a resource account from signer to manage User share Account
        let avs_signer = &avs_manager::get_signer();
        let (registry_coordinator_signer, signer_cap) = account::create_resource_account(avs_signer, REGISTRY_COORDINATOR_NAME);
        avs_manager::add_address(string::utf8(REGISTRY_COORDINATOR_NAME), signer::address_of(&registry_coordinator_signer));
        move_to(&registry_coordinator_signer, RegistryCoordinatorConfigs {
            signer_cap,
        });
    }

    public fun create_registry_coordinator_store() acquires RegistryCoordinatorConfigs{
        let registry_coordinator_signer = registry_coordinator_signer();
        move_to(registry_coordinator_signer, RegistryCoordinatorStore{
            quorum_count: 0,
            quorum_params: smart_table::new(),
            operator_infos: smart_table::new(),
            operator_addresses: smart_table::new(),
            operator_bitmap: smart_table::new(),
            operator_bitmap_history: smart_table::new(),
        })
    }

    #[view]
    public fun is_initialized(): bool{
        avs_manager::address_exists(string::utf8(REGISTRY_COORDINATOR_NAME))
    }

    fun ensure_registry_coordinator_store() acquires RegistryCoordinatorConfigs{
        if(!exists<RegistryCoordinatorStore>(registry_coordinator_address())){
            create_registry_coordinator_store();
        }
    }

    // TODO: not done
    public entry fun registor_operator(operator: &signer, quorum_numbers: vector<u8>, signature: vector<u8>, pubkey: vector<u8>, pop: vector<u8>) acquires RegistryCoordinatorStore{
        let operator_id = get_or_create_operator_id(operator, signature, pubkey, pop);

        let (_ , _ , num_operators_per_quorum) = register_operator_internal(operator, operator_id, quorum_numbers);

        let quorum_numbers_length = vector::length(&quorum_numbers);


        // TODO: limit num operators per quorum
        return
    }

    fun register_operator_internal(operator: &signer, operator_id: vector<u8>, quorum_numbers: vector<u8>): (vector<u128>, vector<u128>, vector<u32>) acquires RegistryCoordinatorStore {
        // TODO: using orderedBytesArrayToBitmap
        let quorum_to_add = math_utils::bytes32_to_u256(quorum_numbers);
        let current_bitmap = current_operator_bitmap(operator_id); 
        // TODO: error name
        assert!(quorum_to_add!=0, 301);
        // TODO: assert no bit in common
        let new_bitmap = current_bitmap | quorum_to_add;
        update_operator_bitmap(operator_id, new_bitmap);

        let mut_store = mut_registry_coordinator_store();
        let operator_address = signer::address_of(operator);
        
        let mut_operator_info = smart_table::borrow_mut_with_default(&mut mut_store.operator_infos, operator_address, OperatorInfo{
            operator_id: operator_id,
            operator_status: 0
        });
        assert!(!smart_table::contains(&mut_store.operator_addresses, operator_id), 1111);
        smart_table::add(&mut mut_store.operator_addresses, operator_id, operator_address);

        if (mut_operator_info.operator_status != 1) {
            mut_operator_info.operator_status = 1;
            // TODO: 
            if(!operator_manager::operator_store_exists(operator_address)){
                operator_manager::create_operator_store(operator_address);
            };
        };

        bls_apk_registry::register_operator(operator, quorum_numbers);

        let (operator_stakes, total_stakes) = stake_registry::register_operator(operator_address, operator_id, quorum_numbers);

        let num_operators_per_quorum = index_registry::register_operator(operator_id, quorum_numbers);
        return (operator_stakes, total_stakes, num_operators_per_quorum)
    }

    public entry fun deregister_operator(operator: &signer, quorum_numbers: vector<u8>) acquires RegistryCoordinatorStore{
        let operator_address = signer::address_of(operator);
        deregister_operator_internal(operator_address, quorum_numbers);
    }

    fun deregister_operator_internal(operator: address, quorum_numbers: vector<u8>) acquires RegistryCoordinatorStore {
        let store = registry_coordinator_store();
        let operator_info = smart_table::borrow(&store.operator_infos, operator);
        let operator_id = operator_info.operator_id;
        assert!(operator_info.operator_status == 1, 202);

        let quorums_to_remove = ordered_vecu8_to_bitmap(quorum_numbers);

        let current_bitmap = current_operator_bitmap(operator_id);

        // TODO: assert here
        let new_bitmap = current_bitmap&(0xff^quorums_to_remove);
        update_operator_bitmap(operator_id, new_bitmap);


        let mut_store = mut_registry_coordinator_store();
        let mut_operator_info = smart_table::borrow_mut(&mut mut_store.operator_infos, operator);
        if (new_bitmap == 0) {
            mut_operator_info.operator_status = 2;
            // TODO: serviceManager.deregisterOperatorFromAVS(operator);

        };

        bls_apk_registry::deregister_operator(operator, quorum_numbers);
        stake_registry::deregister_operator(operator_id, quorum_numbers);
        index_registry::deregister_operator(operator_id, quorum_numbers);
    }

    public entry fun update_operators_for_quorum(
        aggregator: &signer,
        quorum_numbers: vector<u8>,
        opertors_per_quorum: vector<vector<address>>,
    ) acquires RegistryCoordinatorStore {
        assert!(vector::length(&quorum_numbers) == vector::length(&opertors_per_quorum), 105);

        for (i in 0..vector::length(&quorum_numbers)) {
            let quorum_number = *vector::borrow(&quorum_numbers, i);
            let current_quorum_operators = *vector::borrow(&opertors_per_quorum, i);
            let quorum_operator_count = index_registry::quorum_operator_count(quorum_number);
            assert!(vector::length(&current_quorum_operators) == (quorum_operator_count as u64), 106);
            for (j in 0..vector::length(&current_quorum_operators)) {
                let operator_address = *vector::borrow(&current_quorum_operators, j);
                let store = registry_coordinator_store();
        
                let operator_info = smart_table::borrow(&store.operator_infos, operator_address);
                let operator_id = operator_info.operator_id;
                let current_bitmap: u256 = 0;
                let operator_bitmap_history_length = vector::length(smart_table::borrow_with_default(&store.operator_bitmap_history, operator_id, &vector::empty()));
                if (operator_bitmap_history_length != 0) {
                    current_bitmap = vector::borrow(smart_table::borrow(&store.operator_bitmap_history, operator_id), operator_bitmap_history_length-1).quorum_bitmap
                };

                assert!(1 == (current_bitmap >> (quorum_number-1)) & 1, 107);

                if (operator_info.operator_status != 1) {
                    continue
                };

                let store_mut = mut_registry_coordinator_store();
                let operator_info_mut = smart_table::borrow_mut_with_default(&mut store_mut.operator_bitmap_history, operator_id, vector::empty());
                vector::push_back(operator_info_mut,  QuorumBitmapUpdate{
                    update_timestamp: timestamp::now_seconds(),
                    next_update_timestamp: 0,
                    quorum_bitmap: current_bitmap,
                });

                let quorum_to_remove: u256 = stake_registry::update_operator_stake(operator_address, operator_id, vector::singleton(quorum_number));

                if (quorum_to_remove != 0) {

                    deregister_operator_internal(operator_address, bitmap_to_vecu8(quorum_to_remove));
                }
            };
        }
    }


    // TODO: only owner
    public entry fun create_quorum(owner:&signer, max_operator_count: u32 , minumum_stake: u128, strategies: vector<address>, multipliers: vector<u128>) acquires RegistryCoordinatorConfigs, RegistryCoordinatorStore {
        avs_manager::only_owner(signer::address_of(owner));
        ensure_registry_coordinator_store();
        let operator_set_param = OperatorSetParam {
            max_operator_count
        };

        // let strategy = object::address_to_object<Metadata>(strategy_address);
        // let strategy_params = vector::singleton<stake_registry::StrategyParams>(stake_registry::strategy_params(strategy, multiplier));
        
        let strategies_length = vector::length(&strategies);
        let multiplier_length = vector::length(&multipliers);
        
        assert!(strategies_length == multiplier_length , 105);
        let strategy_params = vector::empty<stake_registry::StrategyParams>();
        for (i in 0..strategies_length) {
            let strategy_address = *vector::borrow(&strategies , i);
            let strategy = object::address_to_object<Metadata>(strategy_address);
            let multiplier = *vector::borrow(&multipliers , i);
            vector::push_back(&mut strategy_params, stake_registry::strategy_params(strategy, multiplier));
        };
        create_quorum_internal(operator_set_param, minumum_stake, strategy_params);
    }

    fun create_quorum_internal(operator_set_params: OperatorSetParam, minumum_stake: u128, strategy_params: vector<stake_registry::StrategyParams>) acquires RegistryCoordinatorStore {
        let pre_quorum_count = quorum_count();
        let mut_store = mut_registry_coordinator_store();
        let mut_quorum_count = &mut mut_store.quorum_count;
        *mut_quorum_count = *mut_quorum_count + 1;

        set_operator_set_params_internal(pre_quorum_count+1, operator_set_params);
        stake_registry::initialize_quorum(pre_quorum_count+1, minumum_stake, strategy_params);
        index_registry::initialize_quorum(pre_quorum_count+1);
        bls_apk_registry::initialize_quorum(pre_quorum_count+1);
    }

    public entry fun set_operator_set_params(owner: &signer, quorum_number: u8, max_operator_count: u32) acquires RegistryCoordinatorStore {
        avs_manager::only_owner(signer::address_of(owner));
        let operator_set_params = OperatorSetParam{
            max_operator_count,
        };

        set_operator_set_params_internal(quorum_number, operator_set_params);
    }

    fun set_operator_set_params_internal(quorum_number: u8, operator_set_params: OperatorSetParam) acquires RegistryCoordinatorStore {
        let mut_store = mut_registry_coordinator_store();
        let mut_quorum_param = smart_table::borrow_mut_with_default(&mut mut_store.quorum_params, quorum_number, operator_set_params);
        *mut_quorum_param = operator_set_params
    }

    fun ordered_vecu8_to_bitmap(vec: vector<u8>): u256 {
        let bitmap: u256 = 0;
        let bitmask : u256 = 0;
        let vec_length = vector::length(&vec);
        let first_element = vector::borrow(&vec, 0);
        bitmap = 1 << (*first_element as u8);

        for (i in 1..vec_length) {
            let next_element = vector::borrow(&vec, i);
            bitmask = 1 << *next_element;

            assert!(bitmask > bitmap, 203);
            bitmap = (bitmap | bitmask);
        };
        return bitmap
    }

    fun bitmap_to_vecu8(bitmap: u256): vector<u8> {
        let vecu8: vector<u8> = vector::empty();
        let index = 0;
        let bitmask: u256;
        let  i = 0;
        while (true) {
            bitmask = 1u256 << i;
            if ((bitmap & bitmask) != 0) {
                vector::push_back(&mut vecu8, (i as u8));
                index = index + 1;
            };
            if (i == 255) {
                break
            };
            i = i + 1;
        };
        vecu8
    }

    // TODO: remove public
    fun get_or_create_operator_id(operator: &signer, signature: vector<u8>, pubkey: vector<u8>, pop: vector<u8>): vector<u8>{
        let operator_address = signer::address_of(operator);
        let operator_id = bls_apk_registry::get_operator_id(operator_address);
        if (vector::is_empty(&operator_id)) {
            // TODO: help
            let msg: vector<u8> = vector::empty();
            vector::append(&mut msg, REGISTER_MSG_HASH);
            vector::append(&mut msg, bcs::to_bytes(&operator_address));
            let msg_indentifier = aptos_hash::keccak256(msg);
            operator_id = bls_apk_registry::register_bls_pubkey(operator, signature, pubkey, pop, msg_indentifier);
        };
        return operator_id
    }

    fun pubkey_registration_message_hash(operator: &signer) {
        // TODO: help
    }

    fun current_operator_bitmap(operator_id: vector<u8>):u256 acquires RegistryCoordinatorStore {
        let store = registry_coordinator_store();
        let operator_bitmap_history_length = vector::length(smart_table::borrow_with_default(&store.operator_bitmap_history, operator_id, &vector::empty()));
        if (operator_bitmap_history_length == 0) {
            return 0
        } else {
            return vector::borrow(smart_table::borrow(&store.operator_bitmap_history, operator_id), operator_bitmap_history_length-1).quorum_bitmap
        }
    }

    fun update_operator_bitmap(operator_id : vector<u8>, new_bitmap: u256) acquires RegistryCoordinatorStore {
        let mut_store = mut_registry_coordinator_store();
        if (!smart_table::contains(&mut_store.operator_bitmap_history, operator_id)) {
            smart_table::add(&mut mut_store.operator_bitmap_history, operator_id, vector::singleton(QuorumBitmapUpdate{
                update_timestamp: timestamp::now_seconds(),
                next_update_timestamp: 0,
                quorum_bitmap: new_bitmap,
            }));
        } else {
            let mut_operator_bitmap = smart_table::borrow_mut(&mut mut_store.operator_bitmap_history, operator_id);
            let history_length = vector::length(mut_operator_bitmap);
            let last_update = vector::borrow_mut(mut_operator_bitmap, history_length-1);
            if (last_update.update_timestamp == timestamp::now_seconds()) {
                last_update.quorum_bitmap = new_bitmap;
            } else {
                last_update.next_update_timestamp = timestamp::now_seconds();
                vector::push_back(mut_operator_bitmap, QuorumBitmapUpdate{
                    update_timestamp: timestamp::now_seconds(),
                    next_update_timestamp: 0,
                    quorum_bitmap: new_bitmap,
                });
            }
        }
    }

    #[view]
    public fun get_operator_id(operator: address): vector<u8> acquires RegistryCoordinatorStore {
        let store = registry_coordinator_store();
        smart_table::borrow(&store.operator_infos, operator).operator_id
    }

    #[view]
    public fun get_operator_address(operator_id: vector<u8>): address acquires RegistryCoordinatorStore {
        let store = registry_coordinator_store();
        *smart_table::borrow(&store.operator_addresses, operator_id)
    }

    #[view]
    public fun get_operator_status(operator: address): u8 acquires RegistryCoordinatorStore {
        let store = registry_coordinator_store();
        smart_table::borrow_with_default(&store.operator_infos, operator, &OperatorInfo{
            operator_id: vector::empty<u8>(),
            operator_status: 0,
            }).operator_status
    }

    #[view]
    public fun get_quorum_bitmap_by_timestamp(operator_id: vector<u8>, timestamp: u64): u256 acquires RegistryCoordinatorStore {
        let store = registry_coordinator_store();
        let operator_bitmap_history = smart_table::borrow_with_default(&store.operator_bitmap_history, operator_id, &vector::empty());
        let operator_bitmap_history_length = vector::length(operator_bitmap_history);
        for (i in 0..(operator_bitmap_history_length)) {
            let index = operator_bitmap_history_length - i - 1;
            let update_timestamp = vector::borrow(operator_bitmap_history, index).update_timestamp;
            if (update_timestamp < timestamp) {
                return vector::borrow(operator_bitmap_history, index).quorum_bitmap
            }
        };
        assert!(false, 302);
        return 0
    }

    #[view]
    public fun get_operator_bitmap_history_length(operator_id: vector<u8>): u64 acquires RegistryCoordinatorStore {
        let store = registry_coordinator_store();
        let operator_bitmap_history = smart_table::borrow_with_default(&store.operator_bitmap_history, operator_id, &vector::empty());
        let operator_bitmap_history_length = vector::length(operator_bitmap_history);
        return operator_bitmap_history_length
    }

    #[view]
    public fun get_current_quorum_bitmap(operator_id: vector<u8>): u256 acquires RegistryCoordinatorStore {
        let store = registry_coordinator_store();
        *smart_table::borrow(&store.operator_bitmap, operator_id)
    }

    #[view]
    public fun quorum_count(): u8 acquires RegistryCoordinatorStore {
        let store = registry_coordinator_store();
        store.quorum_count
    }

    inline fun registry_coordinator_store(): &RegistryCoordinatorStore acquires RegistryCoordinatorStore{
        borrow_global<RegistryCoordinatorStore>(registry_coordinator_address())
    }

    inline fun mut_registry_coordinator_store(): &mut RegistryCoordinatorStore acquires RegistryCoordinatorStore {
        borrow_global_mut<RegistryCoordinatorStore>(registry_coordinator_address())
    }

    #[view]
    public fun registry_coordinator_address(): address {
        avs_manager::get_address(string::utf8(REGISTRY_COORDINATOR_NAME))
    }

    inline fun registry_coordinator_signer(): &signer acquires RegistryCoordinatorConfigs{
        &account::create_signer_with_capability(&borrow_global<RegistryCoordinatorConfigs>(registry_coordinator_address()).signer_cap)
    }

    public fun operator_set_param(max_operator_count: u32): OperatorSetParam {
        return OperatorSetParam{
            max_operator_count,
        }
    }

    public fun set_bit(number: u256, index: u8): u256 {
        number | (1u256 << index)
    }
}