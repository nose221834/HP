# syntax=docker/dockerfile:1
# check=error=true

ARG RUBY_VERSION=3.4.1
FROM docker.io/library/ruby:$RUBY_VERSION-slim AS base

# Rails app lives here
WORKDIR /rails

RUN apt-get update -qq && \
    apt-get install --no-install-recommends -y \
    build-essential \
    curl \
    git \
    libjemalloc2 \
    libpq-dev \
    libvips \
    pkg-config \
    libyaml-dev && \
    rm -rf /var/lib/apt/lists/* /var/cache/apt/archives/*


ENV RAILS_ENV="development" \
    BUNDLE_DEPLOYMENT="0" \
    BUNDLE_PATH="/usr/local/bundle" \
    BUNDLE_JOBS="4"

COPY ./ /rails

RUN bundle install

# TODO: あとでつくる
ENTRYPOINT ["/rails/bin/docker-entrypoint"]

CMD ["bin/rails", "server", "-b", "0.0.0.0", "-p", "3000"]
