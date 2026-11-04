# Pluschat Calls

Go WebSocket signaling + WebRTC mesh UI for Pluschat.

## Included
- HMAC-SHA256 signed, short-lived tokens; one-time JTI consumption in MySQL.
- WebSocket signaling for peer join/leave, SDP offer/answer, ICE and hangup.
- Browser UI with microphone, camera and screen sharing.
- MySQL schema for the separate `calls` database.

## Install
1. Create MySQL database `calls`; grant the existing `admin` DB user access. Do not put passwords in Git.
2. Run `database/schema.sql` in `calls`.
3. Copy `public/*` into the document root of `xn--b1ajehof.xn--80avljg2a1c.xn--p1ai`.
4. Copy `config.example` to a private environment file and set DB credentials and `CALL_TOKEN_SECRET`. It must exactly match Pluschat's `APP_SECRET`; never commit it.
5. Build on Linux with Go 1.23+: `go mod download && go build -o pluschat-calls ./cmd/server`.
6. Run the binary as a persistent service. Reverse proxy `/ws` (WebSocket upgrade) and `/health` to 127.0.0.1:8090.
7. Check `/health`.

## Token contract
The main Pluschat endpoint `/api/calls/token.php` authenticates the user and checks membership in `conversation_members`. It signs claims `iss,sub,room,conversation_id,type,role,jti,iat,exp` with HMAC-SHA256 using the shared APP_SECRET. TTL is 3 minutes. The browser receives the token in a URL fragment, never a query string.

## Important limits
This first signaling implementation uses peer-to-peer mesh and is suitable only for small rooms (up to 8 peers, subject to bandwidth). It is **not an SFU** and does not yet provide scalable channel broadcasts. Production calls need TURN; large groups/broadcasts need an SFU such as LiveKit or mediasoup. STUN is not a TURN replacement.
