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
      system "make build"
      puts "done"
    end
  end

  desc "Upload the built binary"
  task :upload do
    on roles(:app) do
      execute "rm #{current_path.join("vega")}"
      upload!("vega",  current_path.join("vega"), recursive: false)
    end
  end

  desc "Reset tendermint"
  task :reset_tendermint do
    on roles(:app) do
      begin
        execute "./tendermint unsafe_reset_all"
      rescue => ex
        puts ex.message.red
      end
    end
  end

  desc "Start tendermint"
  task :start_tendermint do
    on roles(:app) do
      execute "nohup ./tendermint node >/tmp/tendermint.log 2>&1 & sleep 1", pty: true
    end
  end

  desc "Start vega"
  task :start do
    on roles(:app) do
      execute "nohup #{current_path.join("vega")} -remove_expired_gtt=true -log_price_levels=false >/tmp/vega.log 2>&1 & sleep 1", pty: true
    end
  end

  desc "Stop vega"
  task :stop do
    on roles(:app) do
      begin
        execute "killall vega; exit 0"
      rescue => ex
        puts ex.message.red
      end
    end
  end

  desc "Stop tendermint"
  task :stop_tendermint do
    on roles(:app) do
      begin
        execute "killall tendermint; exit 0"
      rescue => ex
        puts ex.message.red
      end
    end
  end

  desc "Reset everything - blow away chain data + restart vega and tendermint"
  task :reset_and_restart do
    on roles(:app) do
      invoke("vega:stop")
      invoke("vega:stop_tendermint")
      invoke("vega:reset_tendermint")
      invoke("vega:start")
      invoke("vega:start_tendermint")
    end
  end

  desc "Blow away server data and publish latest checked out code"
  task :full_reset do
    on roles(:app) do
      invoke!("vega:build")
      invoke!("vega:stop")
      invoke!("vega:stop_tendermint")
      invoke!("vega:upload")
      invoke!("vega:reset_tendermint")
      invoke!("vega:start")
      invoke!("vega:start_tendermint")
    end
  end
end

# Default value for :format is :airbrussh.
# set :format, :airbrussh

# You can configure the Airbrussh format using :format_options.
# These are the defaults.
# set :format_options, command_output: true, log_file: "log/capistrano.log", color: :auto, truncate: :auto

# Default value for :pty is false
# set :pty, true

# Default value for :linked_files is []
# append :linked_files, "config/database.yml", "config/secrets.yml"

# Default value for linked_dirs is []
# append :linked_dirs, "log", "tmp/pids", "tmp/cache", "tmp/sockets", "public/system"

# Default value for default_env is {}
# set :default_env, { path: "/opt/ruby/bin:$PATH" }

# Default value for local_user is ENV['USER']
# set :local_user, -> { `git config user.name`.chomp }

# Default value for keep_releases is 5
# set :keep_releases, 5

# Uncomment the following to require manually verifying the host key before first deploy.
# set :ssh_options, verify_host_key: :secure
