module avs::bls_sig_checker{
    use aptos_framework::account::{Self, SignerCapability};
    use aptos_framework::event;
    use aptos_framework::timestamp;

    use aptos_std::crypto_algebra;
    use aptos_std::bls12381::{Self, PublicKeyWithPoP, AggrPublicKeysWithPoP, AggrOrMultiSignature};
    use aptos_std::bls12381_algebra::{Self, G1, FormatG1Uncompr, HashG1XmdSha256SswuRo};

    use std::string::{Self, String};
    use std::vector;
    use std::signer;

    use restaking::withdrawal;
    use avs::bls_apk_registry; 
    use avs::math_utils;
    use avs::avs_manager; 
    use avs::stake_registry;
    use avs::registry_coordinator;

    const BLS_SIG_CHECKER_NAME: vector<u8> = b"BLS_SIG_CHECKER_NAME";

    const DST: vector<u8> = b"QUUX-V01-CS02-with-BLS12381G1_XMD:SHA-256_SSWU_RO_";

    // Verify signature error
    const EEMPTY_QUORUM: u64 = 1110;
    const EINPUT_QUORUM_LENGTH_MISMATCH: u64 = 1111;
    const ENONSIGNER_LENGTH_MISMATCH: u64 = 1112;
    const EINVALID_TIMESTAMP: u64 = 1113;
    const EQUORUM_APK_HASH_MISMATCH: u64 = 1114;
    const ESIGNATURE_VALIDATE_INVALID: u64 = 1115;

    struct BLSSigCheckerConfig has key {
        signer_cap: SignerCapability,
    }

    public entry fun initialize() {
        if (is_initialized()) {
            return
        };

        // derive a resource account from signer to manage User share Account
        let avs_signer = &avs_manager::get_signer();
        let (bls_sig_checker_signer, signer_cap) = account::create_resource_account(avs_signer, BLS_SIG_CHECKER_NAME);
        avs_manager::add_address(string::utf8(BLS_SIG_CHECKER_NAME), signer::address_of(&bls_sig_checker_signer));
        move_to(&bls_sig_checker_signer, BLSSigCheckerConfig {
            signer_cap,
        });
    }

    #[view]
    /// Return the address of the resource account that stores pool manager configs.
    public fun bls_sig_checker_address(): address {
        avs_manager::get_address(string::utf8(BLS_SIG_CHECKER_NAME))
    }

    #[view]
    public fun is_initialized(): bool{
        avs_manager::address_exists(string::utf8(BLS_SIG_CHECKER_NAME))
    }

    #[view]
    public fun check_signatures(
        quorum_numbers: vector<u8>, 
        reference_timestamp: u64, 
        msg_hashes: vector<vector<u8>>, 
        signer_pubkeys: vector<vector<u8>>,
        signer_sigs: vector<vector<u8>>,
    ): (vector<u128>, vector<u128>) {
        let quorum_length = vector::length(&quorum_numbers);
        assert!(quorum_length > 0, EEMPTY_QUORUM);
        let signer_pubkeys_length = vector::length(&signer_pubkeys);

        let now = timestamp::now_seconds();
        assert!(reference_timestamp < now, EINVALID_TIMESTAMP);

        let signed_stake_for_quorum: vector<u128> = vector::empty();
        let total_stake_for_quorum: vector<u128> = vector::empty();
        let quorum_bitmaps: vector<u256> = vector::empty();
        let pubkey_hashes: vector<vector<u8>> = vector::empty();

        for (i in 0..(signer_pubkeys_length)) {
            let signer_pubkey = *vector::borrow(&signer_pubkeys, i);
            let signer_sig = *vector::borrow(&signer_sigs, i);
            let msg_hash = *vector::borrow(&msg_hashes, i);
            let pubkey_hash = crypto_algebra::hash_to<G1, HashG1XmdSha256SswuRo>(&DST, &signer_pubkey);
            let serialize_pk_hash = crypto_algebra::serialize<G1, FormatG1Uncompr>(&pubkey_hash);
            
            vector::push_back(&mut pubkey_hashes, serialize_pk_hash);
            // TODO: change to specific timestamp
            vector::push_back(&mut quorum_bitmaps, registry_coordinator::get_quorum_bitmap_by_timestamp(serialize_pk_hash, reference_timestamp));
            assert!(bls_apk_registry::validate_signature(serialize_pk_hash, signer_sig, msg_hash), ESIGNATURE_VALIDATE_INVALID);
        
        };

        let withdrawal_delay = withdrawal::minimum_withdrawal_delay();

        for (i in 0..(quorum_length)) {
            // TODO: registryCoordinator.quorumUpdateBlockNumber

            let quorum_number = *vector::borrow(&quorum_numbers, i);
            let total_stake_quorum = stake_registry::total_stake_at_timestamp(
                quorum_number, 
                reference_timestamp
            );
            vector::push_back(&mut total_stake_for_quorum, total_stake_quorum);
            vector::push_back(&mut signed_stake_for_quorum, 0);

            let signer_quorum_index: u64 = 0;
            for (j in 0..(signer_pubkeys_length)) {
                let quorum_bitmap = *vector::borrow(&quorum_bitmaps, j);
                if (1 == (quorum_bitmap >>( quorum_number - 1)) & 1) {
                    let signed_stake = vector::borrow_mut(&mut signed_stake_for_quorum, i);
                    let operator_id = vector::borrow(&mut pubkey_hashes, j);
                    *signed_stake = *signed_stake + stake_registry::get_stake_at_timestamp(
                        quorum_number, 
                        reference_timestamp, 
                        *operator_id
                    );
                    signer_quorum_index = signer_quorum_index + 1;
                }
            }
        };

        return (signed_stake_for_quorum, total_stake_for_quorum)
    }

    inline fun bls_sig_checker_signer(): &signer acquires BLSSigCheckerConfig{
        &account::create_signer_with_capability(&borrow_global<BLSSigCheckerConfig>(bls_sig_checker_address()).signer_cap)
    }
}