# frozen_string_literal: true

require "bundler/setup"
Bundler.require

class RubyE2EApp < Sinatra::Base
  set :bind, "0.0.0.0"
  set :port, ENV.fetch("PORT", "8080")
  set :server, :webrick

  get "/rolldice" do
    rand(1..6).to_s
  end
end

RubyE2EApp.run!
