module ApplicationCable
  class Connection < ActionCable::Connection::Base
    identified_by :connection_identifier

    def connect
      self.connection_identifier = SecureRandom.uuid
        # 接続確立時にセッションIDをクライアントに送信
        transmit({
          type: 'session_id',
          session_id: connection_identifier
        })
        Rails.logger.info "接続確立: #{connection_identifier}"
    end
  end
end
