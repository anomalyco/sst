require_relative "config/environment"
require "json"

def handler(event:, context:)
  puts "Event: #{event.inspect}"
  
  puts "Running migrations..."
  
  # Ensure Rails logs to stdout
  Rails.logger = Logger.new(STDOUT)
  
  # Run migrations via ActiveRecord
  ActiveRecord::Tasks::DatabaseTasks.migrate
  
  puts "Migrations complete!"
  
  {
    statusCode: 200,
    body: JSON.generate({ message: "Migrations completed successfully" })
  }
rescue => e
  puts "Migration failed: #{e.message}"
  puts e.backtrace.join("\n")
  {
    statusCode: 500,
    body: JSON.generate({ error: e.message, backtrace: e.backtrace })
  }
end
