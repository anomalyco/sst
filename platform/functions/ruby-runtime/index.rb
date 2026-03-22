require "net/http"
require "json"
require "time"

def report_error(ex, request_id = nil)
  runtime_api = ENV["AWS_LAMBDA_RUNTIME_API"]
  error_response = {
    errorMessage: ex.message,
    errorType: ex.class.name,
    stackTrace: ex.backtrace
  }

  endpoint = if request_id.nil?
               "http://#{runtime_api}/2018-06-01/runtime/init/error"
             else
               "http://#{runtime_api}/2018-06-01/runtime/invocation/#{request_id}/error"
             end

  uri = URI(endpoint)
  Net::HTTP.post(uri, error_response.to_json, "Content-Type" => "application/json")
end

def log(message)
  puts message
  $stdout.flush
  $stderr.flush
end

handler = ARGV[0] # Expecting "file.method"
runtime_api = ENV["AWS_LAMBDA_RUNTIME_API"]

begin
  file_name, method_name = handler.split(".")
  
  # Add current directory to load path
  $LOAD_PATH.unshift(Dir.pwd) unless $LOAD_PATH.include?(Dir.pwd)
  
  require file_name
rescue Exception => ex
  report_error(ex)
  exit 1
end

loop do
  begin
    # Get next invocation
    next_uri = URI("http://#{runtime_api}/2018-06-01/runtime/invocation/next")
    response = Net::HTTP.get_response(next_uri)
    
    request_id = response["Lambda-Runtime-Aws-Request-Id"]
    deadline_ms = response["Lambda-Runtime-Deadline-Ms"].to_i
    
    context = {
      aws_request_id: request_id,
      invoked_function_arn: response["Lambda-Runtime-Invoked-Function-Arn"],
      get_remaining_time_in_millis: -> { [deadline_ms - (Time.now.to_f * 1000).to_i, 0].max },
      function_name: ENV["AWS_LAMBDA_FUNCTION_NAME"],
      function_version: ENV["AWS_LAMBDA_FUNCTION_VERSION"],
      memory_limit_in_mb: ENV["AWS_LAMBDA_FUNCTION_MEMORY_SIZE"],
      log_group_name: ENV["AWS_LAMBDA_LOG_GROUP_NAME"],
      log_stream_name: ENV["AWS_LAMBDA_LOG_STREAM_NAME"]
    }
    
    event = JSON.parse(response.body)
  rescue Exception => ex
    log("Error getting next invocation: #{ex}")
    report_error(ex)
    next
  end

  # Run the handler function
  begin
    # The handler can be a top-level method or defined in a module if the file required it
    # Most Lambda Ruby handlers are defined as standalone methods in the file
    result = send(method_name, event: event, context: context)
  rescue Exception => ex
    log("Error running handler: #{ex}")
    report_error(ex, request_id)
    next
  end

  # Send the response back to Lambda
  loop do
    begin
      response_uri = URI("http://#{runtime_api}/2018-06-01/runtime/invocation/#{request_id}/response")
      Net::HTTP.post(response_uri, result.to_json, "Content-Type" => "application/json")
      break
    rescue Exception => ex
      log("Error sending response: #{ex}")
      sleep 0.5
      next
    end
  end

  $stdout.flush
  $stderr.flush
end
