module ApplicationCable
  class Connection < ActionCable::Connection::Base
    identified_by :connection_identifier

    # websocket通信を一意ごとにわけるために生成している
    def connect
      self.connection_identifier = SecureRandom.uuid
    end
  end
end
