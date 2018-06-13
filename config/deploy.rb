require "colorize"

lock "~> 3.11.0"

set :application, "vega"
set :deploy_to, "/root/vega"

namespace :vega do
  desc "Builds the vega binary locally"
  task :build do
    run_locally do
      puts "Building vega binary..."
      system "go build"
      puts "done"
    end
  end

  desc "Reset tendermint"
  task :reset_tendermint do
    on roles(:app) do
      begin
       execute "docker exec -i -t vega tendermint unsafe_reset_all"
       execute "/root/vega/message.sh 'Resetting tendermint'"
      rescue => ex
       puts ex.message.red
     end
   end
  end

  desc "Start vega & tendermint"
  task :start do
    on roles(:app) do
      begin
        execute "/root/vega/startup.sh"
      rescue => ex
        puts ex.message.red
      end
    end
  end

  desc "Restart vega & tendermint"
  task :restart do
    on roles(:app) do
      begin
        execute "/root/vega/message.sh 'Restarting vega & tendermint'"
        execute "docker restart vega"
      rescue => ex
        puts ex.message.red
      end
    end
  end

  desc "Stop vega & tendermint"
  task :stop do
    on roles(:app) do
      begin
        execute "/root/vega/message.sh 'Stopping vega & tendermint'"
        execute "docker stop vega"
      rescue => ex
        puts ex.message.red
      end
    end
  end

  desc "Update vega & tendermint"
  task :stop do
    on roles(:app) do
      begin
        execute "/root/vega/message.sh 'Updating vega & tendermint'"
        execute "docker stop vega"
        execute "/root/vega/update.sh"
        invoke("vega:start")
      rescue => ex
        puts ex.message.red
      end
    end
  end

  desc "Reset everything - blow away chain data + restart vega and tendermint"
  task :reset_app_servers do
    on roles(:app) do
      invoke("vega:reset_tendermint")
      invoke("vega:restart")
    end
  end

  desc "Blow away server data and publish latest checked out code"
  task :full_reset do
    on roles(:app) do
      invoke!("vega:reset_tendermint")
      invoke!("vega:update")
    end
  end

  desc "Uptime in Slack"
  task :uptime do
    on roles(:app) do
      begin
        execute "uptime | xargs -0 /root/vega/message.sh"
      rescue => ex
        puts ex.message.red
      end
    end
  end
end
