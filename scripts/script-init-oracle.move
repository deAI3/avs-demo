script {
  fun initialize_avs_modules() {
    avs::service_manager::initialize();
    avs::service_manager_base::initialize();
    avs::bls_apk_registry::initialize();
    avs::bls_sig_checker::initialize();
    avs::index_registry::initialize();
    avs::stake_registry::initialize();
    avs::registry_coordinator::initialize();
    avs::registry_coordinator::create_registry_coordinator_store();
    avs::bls_apk_registry::create_bls_apk_registry_store();
    avs::index_registry::create_index_registry_store();
    avs::stake_registry::create_stake_regsitry_store();
  }
}