# frozen_string_literal: true

# Copyright The OpenTelemetry Authors
#
# SPDX-License-Identifier: Apache-2.0

# OTelBundlerPatch
module OTelBundlerPatch
  # ref: https://github.com/newrelic/newrelic-ruby-agent/blob/dev/lib/boot/strap.rb
  def require(*_groups)
    super
    require_otel
  end

  # this is used for case when user particularly want to enable single instrumentation
  OTEL_INSTRUMENTATION_MAP = {
    'gruf' => 'OpenTelemetry::Instrumentation::Gruf',
    'trilogy' => 'OpenTelemetry::Instrumentation::Trilogy',
    'active_support' => 'OpenTelemetry::Instrumentation::ActiveSupport',
    'action_pack' => 'OpenTelemetry::Instrumentation::ActionPack',
    'active_job' => 'OpenTelemetry::Instrumentation::ActiveJob',
    'active_record' => 'OpenTelemetry::Instrumentation::ActiveRecord',
    'action_view' => 'OpenTelemetry::Instrumentation::ActionView',
    'action_mailer' => 'OpenTelemetry::Instrumentation::ActionMailer',
    'aws_sdk' => 'OpenTelemetry::Instrumentation::AwsSdk',
    'aws_lambda' => 'OpenTelemetry::Instrumentation::AwsLambda',
    'bunny' => 'OpenTelemetry::Instrumentation::Bunny',
    'lmdb' => 'OpenTelemetry::Instrumentation::LMDB',
    'http' => 'OpenTelemetry::Instrumentation::HTTP',
    'koala' => 'OpenTelemetry::Instrumentation::Koala',
    'active_model_serializers' => 'OpenTelemetry::Instrumentation::ActiveModelSerializers',
    'concurrent_ruby' => 'OpenTelemetry::Instrumentation::ConcurrentRuby',
    'dalli' => 'OpenTelemetry::Instrumentation::Dalli',
    'delayed_job' => 'OpenTelemetry::Instrumentation::DelayedJob',
    'ethon' => 'OpenTelemetry::Instrumentation::Ethon',
    'excon' => 'OpenTelemetry::Instrumentation::Excon',
    'faraday' => 'OpenTelemetry::Instrumentation::Faraday',
    'grape' => 'OpenTelemetry::Instrumentation::Grape',
    'graphql' => 'OpenTelemetry::Instrumentation::GraphQL',
    'http_client' => 'OpenTelemetry::Instrumentation::HttpClient',
    'mongo' => 'OpenTelemetry::Instrumentation::Mongo',
    'mysql2' => 'OpenTelemetry::Instrumentation::Mysql2',
    'net_http' => 'OpenTelemetry::Instrumentation::Net::HTTP',
    'pg' => 'OpenTelemetry::Instrumentation::PG',
    'que' => 'OpenTelemetry::Instrumentation::Que',
    'racecar' => 'OpenTelemetry::Instrumentation::Racecar',
    'rack' => 'OpenTelemetry::Instrumentation::Rack',
    'rails' => 'OpenTelemetry::Instrumentation::Rails',
    'rake' => 'OpenTelemetry::Instrumentation::Rake',
    'rdkafka' => 'OpenTelemetry::Instrumentation::Rdkafka',
    'redis' => 'OpenTelemetry::Instrumentation::Redis',
    'restclient' => 'OpenTelemetry::Instrumentation::RestClient',
    'resque' => 'OpenTelemetry::Instrumentation::Resque',
    'ruby_kafka' => 'OpenTelemetry::Instrumentation::RubyKafka',
    'sidekiq' => 'OpenTelemetry::Instrumentation::Sidekiq',
    'sinatra' => 'OpenTelemetry::Instrumentation::Sinatra'
  }.freeze

  def detect_resource_from_env
    env = ENV['OTEL_RUBY_RESOURCE_DETECTORS'].to_s
    additional_resource = ::OpenTelemetry::SDK::Resources::Resource.create({})

    env.split(',').each do |detector|
      case detector
      when 'container'
        additional_resource = additional_resource.merge(::OpenTelemetry::Resource::Detector::Container.detect) if defined? ::OpenTelemetry::Resource::Detector::Container
      when 'google_cloud_platform'
        additional_resource = additional_resource.merge(::OpenTelemetry::Resource::Detector::GoogleCloudPlatform.detect) if defined? ::OpenTelemetry::Resource::Detector::GoogleCloudPlatform
      when 'azure'
        additional_resource = additional_resource.merge(::OpenTelemetry::Resource::Detector::Azure.detect) if defined? ::OpenTelemetry::Resource::Detector::Azure
      end
    end

    additional_resource
  end

  def determine_enabled_instrumentation
    env = ENV['OTEL_RUBY_ENABLED_INSTRUMENTATIONS'].to_s

    env.split(',').map { |instrumentation| OTEL_INSTRUMENTATION_MAP[instrumentation] }
  end

  def require_otel
    lib = File.expand_path('..', __dir__)
    $LOAD_PATH.reject! { |path| path.include?('zero-code-instrumentation') }
    $LOAD_PATH.unshift(lib)

    begin
      required_instrumentation = determine_enabled_instrumentation

      OpenTelemetry::SDK.configure do |c|
        c.resource = detect_resource_from_env
        if required_instrumentation.empty?
          c.use_all # enables all instrumentation!
        else
          required_instrumentation.each do |instrumentation|
            c.use instrumentation
          end
        end
      end
      OpenTelemetry.logger.info { 'Auto-instrumentation initialized' }
    rescue StandardError => e
      OpenTelemetry.logger.info { "Auto-instrumentation failed to initialize. Error: #{e.message}" }
    end
  end
end

require 'bundler'

container = ENV['OTEL_RUBY_RESOURCE_DETECTORS'].to_s.include?('container')
google_cloud_platform = ENV['OTEL_RUBY_RESOURCE_DETECTORS'].to_s.include?('google_cloud_platform')
azure = ENV['OTEL_RUBY_RESOURCE_DETECTORS'].to_s.include?('azure')

# set OTEL_OPERATOR to true if in autoinstrumentation-ruby image
# /otel-auto-instrumentation-ruby is set in operator ruby.go
operator_gem_path = ENV['OTEL_OPERATOR'].to_s == 'true' ? '/otel-auto-instrumentation-ruby' : nil
additional_gem_path = operator_gem_path || ENV['ADDITIONAL_GEM_PATH'] || Gem.dir
puts "Loading the additional gem path from #{additional_gem_path}"

# google-protobuf is used for otel trace exporter
Dir.glob("#{additional_gem_path}/gems/*").each do |file|
  if file.include?('opentelemetry') || file.include?('google')
    puts "Unshift #{file.inspect}"
    $LOAD_PATH.unshift("#{file}/lib")
  end
end

require 'opentelemetry-sdk'
require 'opentelemetry-instrumentation-all'
require 'opentelemetry-helpers-mysql'
require 'opentelemetry-helpers-sql-obfuscation'
require 'opentelemetry-exporter-otlp'

require 'opentelemetry-resource-detector-container' if container
require 'opentelemetry-resource-detector-google_cloud_platform' if google_cloud_platform
require 'opentelemetry-resource-detector-azure' if azure

Bundler::Runtime.prepend(OTelBundlerPatch)

Bundler.require if ENV['REQUIRE_BUNDLER'].to_s == 'true'
