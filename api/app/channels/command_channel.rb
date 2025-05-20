class CommandChannel < ApplicationCable::Channel
  def subscribed
    stream_from "command_channel_#{connection.connection_identifier}"
  end

  def unsubscribed
    # クリーンアップ処理
  end

  def execute_command(data)
    return unless data['command'].present?

    result = CommandExecutorService.execute(data['command'])
    
    # 結果を特定のクライアントにのみ送信
    ActionCable.server.broadcast(
      "command_channel_#{connection.connection_identifier}",
      result
    )
  end
end 