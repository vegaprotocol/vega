require "colorize"

# config valid for current version and patch releases of Capistrano
#lock "~> 3.11.0"

set :application, "vega"
# set :repo_url, "git@gitlab.com:vega-protocol/trading-core.git"

# Default branch is :master
# ask :branch, `git rev-parse --abbrev-ref HEAD`.chomp

# Default deploy_to directory is /var/www/my_app_name
set :deploy_to, "/home/vega/"

namespace :vega do
  desc "Builds the vega binary locally"
  task :build do
    run_locally do
      puts "Building vega binary..."
      system "make build -C ../"
      puts "done"
    end
  end

  desc "Upload the built binary"
  task :upload do
    on roles(:app) do
      #execute "rm #{current_path.join("vega")}"
      upload!("../vega",  current_path.join("vega"), recursive: false)
    end
  end
end
