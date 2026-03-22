# Specify the Ruby version as an ARG
ARG RUBY_VERSION=3.3
ARG RUBY_RUNTIME

# Stage 1: Build environment (install build tools and dependencies)
FROM public.ecr.aws/lambda/ruby:${RUBY_VERSION} AS build

# Ensure git and build tools are installed
RUN yum install -y git gcc make gcc-c++

# Copy Gemfile and Gemfile.lock (if it exists)
COPY Gemfile* ${LAMBDA_TASK_ROOT}/

# Install dependencies using Bundler
WORKDIR ${LAMBDA_TASK_ROOT}
RUN bundle config set --local deployment 'true' && \
    bundle config set --local path 'vendor/bundle' && \
    bundle install

# Stage 2: Final runtime image
FROM public.ecr.aws/lambda/ruby:${RUBY_VERSION}

# Copy the installed dependencies from the build stage
COPY --from=build ${LAMBDA_TASK_ROOT}/vendor/bundle ${LAMBDA_TASK_ROOT}/vendor/bundle
COPY --from=build ${LAMBDA_TASK_ROOT}/.bundle ${LAMBDA_TASK_ROOT}/.bundle

# Copy the application code into the final image
COPY . ${LAMBDA_TASK_ROOT}

# Ensure dependencies are available in the load path
ENV BUNDLE_PATH=${LAMBDA_TASK_ROOT}/vendor/bundle
ENV BUNDLE_WITHOUT=development:test
ENV GEM_PATH=${LAMBDA_TASK_ROOT}/vendor/bundle/ruby/3.3.0

# No need to configure the handler or entrypoint - SST will do that
