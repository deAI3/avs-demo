module avs::bls_apk_registry{
    use aptos_framework::account::{Self, SignerCapability};
    use aptos_framework::event;
    use aptos_framework::timestamp;

    use aptos_std::crypto_algebra;
    use aptos_std::bls12381::{AggrPublicKeysWithPoP, Signature, PublicKeyWithPoP, aggregate_pubkeys, aggregate_pubkey_to_bytes, signature_from_bytes, proof_of_possession_from_bytes, proof_of_possession_to_bytes, public_key_from_bytes_with_pop, public_key_with_pop_to_bytes, public_key_with_pop_to_normal, verify_signature_share, verify_normal_signature};
    use aptos_std::bls12381_algebra::{G1, HashG1XmdSha256SswuRo, FormatG1Uncompr};
    use aptos_std::smart_table::{Self, SmartTable};
    use aptos_std::option::{Self, Option};

    use std::string::{Self, String};
    use std::vector;
    use std::signer;
    
    use avs::avs_manager; 
    use avs::registry_coordinator;

    friend avs::registry_coordinator;

    const ZERO_PK_HASH: vector<u8> = x"ad3228b676f7d3cd4284a5443f17f1962b36e491b30a40b2405849e597ba5fb5";
    const DST: vector<u8> = b"QUUX-V01-CS02-with-BLS12381G1_XMD:SHA-256_SSWU_RO_";
    
    const BLS_APK_REGISTRY_NAME: vector<u8> = b"BLS_APK_REGISTRY_NAME";

    const EQUORUM_ALREADY_EXIST: u64 = 1101;
    const EQUORUM_DOES_NOT_EXIST: u64 = 1102;
    const EZERO_PUBKEY: u64 = 1103;
    const EINVALID_PUBKEY_1: u64 = 1104;
    const EINVALID_PUBKEY_2: u64 = 1111;
    const EINVALID_PUBKEY_G2: u64 = 1105;
    const EOPERATOR_ALREADY_EXIST: u64 = 1106;
    const EPUBKEY_ALREADY_EXIST: u64 = 1107;
    const EPUBKEY_NOT_EXIST: u64 = 1108;
    const ESIGNATURE_INVALID: u64 = 1109;
    const EQUORUM_APK_UPDATE_INVALID_INDEX: u64 = 1110;
    const EINVALID_TIMESTAMP: u64 = 1111;

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

    struct BLSApkRegistryConfigs has key {
        signer_cap: SignerCapability,
    }

    public entry fun initialize() {
        if (is_initialized()) {
            return
        };

        // derive a resource account from signer to manage User share Account
        let avs_signer = &avs_manager::get_signer();
        let (bls_apk_registry_signer, signer_cap) = account::create_resource_account(avs_signer, BLS_APK_REGISTRY_NAME);
        avs_manager::add_address(string::utf8(BLS_APK_REGISTRY_NAME), signer::address_of(&bls_apk_registry_signer));
        move_to(&bls_apk_registry_signer, BLSApkRegistryConfigs {
            signer_cap,
        });
    }

    #[view]
    public fun is_initialized(): bool{
        avs_manager::address_exists(string::utf8(BLS_APK_REGISTRY_NAME))
    }

    #[view]
    /// Return the address of the resource account that stores pool manager configs.
    public fun bls_apk_registry_address(): address {
      avs_manager::get_address(string::utf8(BLS_APK_REGISTRY_NAME))
    }

    fun ensure_bls_apk_registry_store() acquires BLSApkRegistryConfigs{
        if(!exists<BLSApkRegistryStore>(bls_apk_registry_address())){
            create_bls_apk_registry_store();
        }
    }

    public fun create_bls_apk_registry_store() acquires BLSApkRegistryConfigs{
        let bls_apk_registry = bls_apk_registry_address();
        let bls_apk_registry_signer = bls_apk_registry_signer();
        move_to(bls_apk_registry_signer, BLSApkRegistryStore{
            operator_to_pk_hash: smart_table::new(),
            pk_hash_to_operator: smart_table::new(),
            operator_to_pk: smart_table::new(),
            apk_history: smart_table::new(),
            current_apk: smart_table::new()
        })
    }


    public(friend) fun initialize_quorum(quorum_number: u8) acquires BLSApkRegistryStore, BLSApkRegistryConfigs{
        ensure_bls_apk_registry_store();
        let store = bls_apk_registry_store_mut();
        assert!(!smart_table::contains(&store.apk_history, quorum_number), EQUORUM_ALREADY_EXIST);
        let apk_history: vector<ApkUpdate> = vector::empty();
        let now = timestamp::now_seconds();
        vector::push_back(&mut apk_history, ApkUpdate{
            aggregate_pubkeys: option::none(),
            update_timestamp: now,
            next_update_timestamp: 0,
        });
        smart_table::add(&mut store.apk_history, quorum_number, apk_history);
        smart_table::add(&mut store.current_apk, quorum_number, vector::empty());
    }

    public(friend) fun register_operator(operator: &signer, quorum_numbers: vector<u8>) acquires BLSApkRegistryStore {
        let operator_address = signer::address_of(operator);
        let store = bls_apk_registry_store();
        let pubkey = smart_table::borrow(&store.operator_to_pk, operator_address);

        update_quorum_apk(quorum_numbers, *pubkey, true)
    }

    public(friend) fun deregister_operator(operator: address, quorum_numbers: vector<u8>) acquires BLSApkRegistryStore {
        let store = bls_apk_registry_store();
        let pubkey = smart_table::borrow(&store.operator_to_pk, operator);

        update_quorum_apk(quorum_numbers, *pubkey, false)
    }

    public(friend) fun register_bls_pubkey(operator: &signer, signature: vector<u8>, pubkey: vector<u8>, pop: vector<u8>, msg: vector<u8>): vector<u8> acquires BLSApkRegistryStore {
        let pop = proof_of_possession_from_bytes(pop);
        let pubkey_with_pop = public_key_from_bytes_with_pop(pubkey, &pop);
        assert!(option::is_some(&pubkey_with_pop), EINVALID_PUBKEY_1);
        let unwrapped_pubkey = option::borrow(&pubkey_with_pop);
        let pubkey_bytes = public_key_with_pop_to_bytes(unwrapped_pubkey);
        assert!(vector::length(&pubkey_bytes) == 48, EINVALID_PUBKEY_2);

        let pubkey_hash = crypto_algebra::hash_to<G1, HashG1XmdSha256SswuRo>(&DST, &pubkey);
        let serialize_pk_hash = crypto_algebra::serialize<G1, FormatG1Uncompr>(&pubkey_hash);

        let store = bls_apk_registry_store();
        let operator_address = signer::address_of(operator);
        assert!(!smart_table::contains(&store.operator_to_pk_hash, operator_address), EOPERATOR_ALREADY_EXIST);
        assert!(!smart_table::contains(&store.pk_hash_to_operator, serialize_pk_hash), EPUBKEY_ALREADY_EXIST);

        assert!(verify_signature_share(&signature_from_bytes(signature), unwrapped_pubkey, msg), ESIGNATURE_INVALID);

        let store_mut = bls_apk_registry_store_mut();
        smart_table::upsert(&mut store_mut.operator_to_pk, operator_address, *option::borrow(&pubkey_with_pop));
        smart_table::upsert(&mut store_mut.operator_to_pk_hash, operator_address, serialize_pk_hash);
        smart_table::upsert(&mut store_mut.pk_hash_to_operator, serialize_pk_hash, operator_address);
        
        // TODO emit event
        return serialize_pk_hash
    }

    fun update_quorum_apk(quorum_numbers: vector<u8>, pubkey: PublicKeyWithPoP, register: bool) acquires BLSApkRegistryStore {
        let i = 0;
        while (i < vector::length(&quorum_numbers)) {
            let quorum_number = *vector::borrow(&quorum_numbers, i);
            let apk_history_length = vector::length(smart_table::borrow_with_default(&bls_apk_registry_store().apk_history, quorum_number, &vector::empty()));
            assert!(apk_history_length > 0, EQUORUM_DOES_NOT_EXIST);

            // Update pubkey
            let current_apk = current_apk_mut(quorum_number);
            if (register) {
                vector::push_back(current_apk, pubkey);
            } else {
                assert!(vector::contains(current_apk, &pubkey), EPUBKEY_NOT_EXIST);
                vector::remove_value(current_apk, &pubkey);
            };

            let borrow_new_apk = *current_apk;
            let new_aggr_pubkeys : AggrPublicKeysWithPoP;
            let option_aggr_pubkeys = option::none();
            if (!vector::is_empty(&borrow_new_apk)) {
                new_aggr_pubkeys = aggregate_pubkeys(borrow_new_apk);
                option_aggr_pubkeys = option::some(new_aggr_pubkeys);
            };
            
            
            let latest_update = latest_apk_update_mut(quorum_number);
            let now = timestamp::now_seconds();
            if (latest_update.update_timestamp == now) {
                latest_update.aggregate_pubkeys = option_aggr_pubkeys;
            } else {
                latest_update.next_update_timestamp = now;
                let store_mut = bls_apk_registry_store_mut();
                let apk_history_mut = smart_table::borrow_mut(&mut store_mut.apk_history, quorum_number);
                vector::push_back(apk_history_mut, ApkUpdate{
                    aggregate_pubkeys: option_aggr_pubkeys,
                    update_timestamp: now,
                    next_update_timestamp: 0
                })
            };
            i = i + 1;
        }
    }

    #[view]
    public fun validate_signature(operator_id: vector<u8>, signature: vector<u8>, msg: vector<u8>): bool acquires BLSApkRegistryStore{
        let store = bls_apk_registry_store();
        if (!smart_table::contains(&store.pk_hash_to_operator, operator_id)) {
            return false
        };
        let operator_address = *smart_table::borrow(&store.pk_hash_to_operator, operator_id);
        if (!smart_table::contains(&store.operator_to_pk, operator_address)) {
            return false
        };
        let pubkey_with_pop = smart_table::borrow(&store.operator_to_pk, operator_address);
        let pubkey = public_key_with_pop_to_normal(pubkey_with_pop);
        return verify_normal_signature(&signature_from_bytes(signature), &pubkey, msg)
    } 

    #[view]
    public fun get_operator_id(operator: address): vector<u8> acquires BLSApkRegistryStore{
        let store = bls_apk_registry_store();
        let operator_id = smart_table::borrow_with_default(&store.operator_to_pk_hash, operator, &vector::empty<u8>());
        return *operator_id
    }

    #[view]
    public fun get_operator_pk(operator: address): PublicKeyWithPoP acquires BLSApkRegistryStore{
        let store = bls_apk_registry_store();
        let pubkey = smart_table::borrow(&store.operator_to_pk, operator);
        return *pubkey
    }

    #[view]
    public fun get_aggr_pk_hash_at_timestamp(quorum_number: u8, timestamp: u64): vector<u8> acquires BLSApkRegistryStore {
        let store = bls_apk_registry_store();
        assert!(smart_table::contains(&store.apk_history, quorum_number), EQUORUM_DOES_NOT_EXIST);
        let quorum_apk_update = smart_table::borrow(&store.apk_history, quorum_number);
        let quorum_apk_update_length = vector::length(quorum_apk_update);

        for (i in 0..(quorum_apk_update_length)) {
            let index = quorum_apk_update_length - i - 1;
            let update_timestamp = vector::borrow(quorum_apk_update, i).update_timestamp;
            if (update_timestamp < timestamp) {
                let aggregate_pubkeys = option::borrow(&vector::borrow(quorum_apk_update, i).aggregate_pubkeys);
                return aggregate_pubkey_to_bytes(aggregate_pubkeys)
            }
        };

        assert!(false, EINVALID_TIMESTAMP);
        return vector::empty()
    }
    
    inline fun latest_apk_update_mut(quorum_number: u8): &mut ApkUpdate acquires BLSApkRegistryStore {
        let store_mut = bls_apk_registry_store_mut();
        let apk_history_length = vector::length(smart_table::borrow(&store_mut.apk_history, quorum_number));
        let apk_history = vector::borrow_mut(smart_table::borrow_mut(&mut store_mut.apk_history, quorum_number), apk_history_length - 1);
        apk_history
    }
    inline fun current_apk_mut(quorum_number: u8): &mut vector<PublicKeyWithPoP> acquires BLSApkRegistryStore {
        let store_mut = bls_apk_registry_store_mut();
        let current_apk_mut = smart_table::borrow_mut(&mut store_mut.current_apk, quorum_number);
        current_apk_mut
    }

    inline fun bls_apk_registry_store(): &BLSApkRegistryStore  acquires BLSApkRegistryStore {
        borrow_global<BLSApkRegistryStore>(bls_apk_registry_address())
    }

    inline fun bls_apk_registry_store_mut(): &mut BLSApkRegistryStore  acquires BLSApkRegistryStore {
        borrow_global_mut<BLSApkRegistryStore>(bls_apk_registry_address())
    }

    inline fun bls_apk_registry_signer(): &signer acquires BLSApkRegistryConfigs{
        &account::create_signer_with_capability(&borrow_global<BLSApkRegistryConfigs>(bls_apk_registry_address()).signer_cap)
    }
}