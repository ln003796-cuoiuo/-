CREATE TABLE IF NOT EXISTS consumed_tokens (
 jti CHAR(36) NOT NULL, user_id BIGINT UNSIGNED NOT NULL, expires_at DATETIME NOT NULL,
 consumed_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP, PRIMARY KEY(jti), KEY idx_consumed_expiry(expires_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE TABLE IF NOT EXISTS call_sessions (
 id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT, room_id VARCHAR(80) NOT NULL,
 conversation_id BIGINT UNSIGNED NOT NULL, call_type ENUM('audio','video','broadcast') NOT NULL,
 started_by BIGINT UNSIGNED NOT NULL, status ENUM('active','ended') NOT NULL DEFAULT 'active',
 started_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP, ended_at DATETIME NULL,
 PRIMARY KEY(id), UNIQUE KEY uq_call_room(room_id), KEY idx_call_conversation(conversation_id,started_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
