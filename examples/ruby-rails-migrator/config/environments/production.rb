require "rails/all"

module RubyRailsMigrator
  class Application < Rails::Application
    config.enable_reloading = false
    config.eager_load = true
    config.consider_all_requests_local = false
    config.active_support.report_deprecations = false
    config.active_record.dump_schema_after_migration = false
  end
end
