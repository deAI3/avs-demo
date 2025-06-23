module avs::fee_pool{
    use aptos_framework::event;
    use aptos_framework::fungible_asset::{
        Self, FungibleAsset, FungibleStore, Metadata,
    };
    use aptos_framework::object::{Self, Object};
    use aptos_framework::primary_fungible_store;
    use std::bcs;
    use std::vector;
    use std::signer;

    use avs::avs_manager;

    friend avs::service_manager;

    struct FeePool has key {
        token_store: Object<FungibleStore>,
    }

    public(friend) fun ensure_fee_pool(token: Object<Metadata>): Object<FeePool> {
        let seeds = get_pool_seeds(token);

        let package_signer = &avs_manager::get_signer();
        let fee_pool_addr = object::create_object_address(&signer::address_of(package_signer), seeds);

        if(object::object_exists<FeePool>(fee_pool_addr)){
            return object::address_to_object<FeePool>(fee_pool_addr)
        };
        let ctor = &object::create_named_object(package_signer, seeds);
        
        let pool_signer = &object::generate_signer(ctor);

        let store = fungible_asset::create_store(ctor, token);

        let pool = FeePool {
            token_store: store,
        };

        move_to(pool_signer, pool);

        object::object_from_constructor_ref(ctor)
    }

    // assume that the asset has already been transferred to the store
    public(friend) fun deposit(pool: Object<FeePool>, fa: FungibleAsset): u64 acquires FeePool {
        let fee_pool = mut_fee_pool(&pool);
        let token = fungible_asset::store_metadata(fee_pool.token_store);
        fungible_asset::deposit(fee_pool.token_store, fa);
        return fungible_asset::balance(fee_pool.token_store)
    }

    public(friend) fun withdraw(pool: Object<FeePool>, recipient: address, amount: u64) acquires FeePool {
        let fee_pool = mut_fee_pool(&pool);

        let token = fungible_asset::store_metadata(fee_pool.token_store);
        let pool_signer = &avs_manager::get_signer();
        let withdrawal = fungible_asset::withdraw(pool_signer, fee_pool.token_store, amount);

        let to = primary_fungible_store::ensure_primary_store_exists(recipient, token);
        fungible_asset::deposit(to, withdrawal);
    }

    #[view]
    public fun token_store(pool: Object<FeePool>): Object<FungibleStore> acquires FeePool {
        fee_pool(&pool).token_store
    }

    #[view]
    public fun token_metadata(pool: Object<FeePool>): Object<Metadata> acquires FeePool {
        fungible_asset::store_metadata(token_store(pool))
    }

    inline fun fee_pool(pool: &Object<FeePool>): &FeePool acquires FeePool {
        borrow_global<FeePool>(object::object_address(pool))
    }

    inline fun mut_fee_pool(pool: &Object<FeePool>): &mut FeePool acquires FeePool {
        borrow_global_mut<FeePool>(object::object_address(pool))
    }

    inline fun get_pool_seeds(token: Object<Metadata>): vector<u8>{
        let seeds = vector[];
        vector::append(&mut seeds, bcs::to_bytes(&object::object_address(&token)));
        seeds
    }
}