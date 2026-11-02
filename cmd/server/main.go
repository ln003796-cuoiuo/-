package main

import (
 "crypto/hmac"
 "crypto/sha256"
 "database/sql"
 "encoding/base64"
 "encoding/json"
 "fmt"
 "log"
 "net/http"
 "os"
 "strconv"
 "strings"
 "sync"
 "time"
 _ "github.com/go-sql-driver/mysql"
 "github.com/gorilla/websocket"
)
type Claims struct { Iss string `json:"iss"`; Sub string `json:"sub"`; Room string `json:"room"`; Conversation int64 `json:"conversation_id"`; Type string `json:"type"`; Role string `json:"role"`; JTI string `json:"jti"`; Exp int64 `json:"exp"`; Iat int64 `json:"iat"` }
type Client struct { conn *websocket.Conn; user,room string; send chan []byte }
type Hub struct { mu sync.RWMutex; rooms map[string]map[*Client]bool; db *sql.DB }
var hub *Hub
var upgrader=websocket.Upgrader{ReadBufferSize:4096,WriteBufferSize:4096,CheckOrigin:func(r *http.Request)bool{origin:=r.Header.Get("Origin");for _,v:=range strings.Split(os.Getenv("CALL_ALLOWED_ORIGINS"),","){if strings.EqualFold(strings.TrimSpace(v),origin)&&origin!=""{return true}};return false}}
func env(k,d string)string{if v:=os.Getenv(k);v!=""{return v};return d}
func verify(token string)(Claims,error){var c Claims;p:=strings.Split(token,".");if len(p)!=3{return c,fmt.Errorf("bad token")};m:=hmac.New(sha256.New,[]byte(env("CALL_TOKEN_SECRET","")));m.Write([]byte(p[0]+"."+p[1]));sig,e:=base64.RawURLEncoding.DecodeString(p[2]);if e!=nil||!hmac.Equal(sig,m.Sum(nil)){return c,fmt.Errorf("bad signature")};raw,e:=base64.RawURLEncoding.DecodeString(p[1]);if e!=nil{return c,e};if e=json.Unmarshal(raw,&c);e!=nil{return c,e};now:=time.Now().Unix();if c.Iss!=env("CALL_TOKEN_ISSUER","pluschat")||c.Sub==""||c.Room==""||c.JTI==""||c.Conversation<1||c.Exp<now||c.Exp>now+600||(c.Type!="audio"&&c.Type!="video"){return c,fmt.Errorf("invalid claims")};return c,nil}
func(h *Hub)join(c *Client)int{h.mu.Lock();defer h.mu.Unlock();if h.rooms[c.room]==nil{h.rooms[c.room]=map[*Client]bool{}};h.rooms[c.room][c]=true;return len(h.rooms[c.room])}
func(h *Hub)leave(c *Client){h.mu.Lock();defer h.mu.Unlock();delete(h.rooms[c.room],c);if len(h.rooms[c.room])==0{delete(h.rooms,c.room)}}
func(h *Hub)peers(c *Client)[]*Client{h.mu.RLock();defer h.mu.RUnlock();a:=[]*Client{};for p:=range h.rooms[c.room]{if p!=c{a=append(a,p)}};return a}
func send(c *Client,v any){b,e:=json.Marshal(v);if e==nil{select{case c.send<-b:default:}}}
func wsHandler(w http.ResponseWriter,r *http.Request){conn,e:=upgrader.Upgrade(w,r,nil);if e!=nil{return};conn.SetReadLimit(1<<20);conn.SetReadDeadline(time.Now().Add(10*time.Second));_,b,e:=conn.ReadMessage();if e!=nil{conn.Close();return};var a struct{Type,Token string};if json.Unmarshal(b,&a)!=nil||a.Type!="auth"{conn.Close();return};cl,e:=verify(a.Token);if e!=nil{_ = conn.WriteJSON(map[string]string{"type":"error","error":"invalid_token"});conn.Close();return}
 _,e=hub.db.Exec("INSERT INTO consumed_tokens(jti,user_id,expires_at) VALUES(?,?,FROM_UNIXTIME(?))",cl.JTI,cl.Sub,cl.Exp);if e!=nil{_ = conn.WriteJSON(map[string]string{"type":"error","error":"token_already_used"});conn.Close();return}
 c:=&Client{conn:conn,user:cl.Sub,room:cl.Room,send:make(chan []byte,64)};n:=hub.join(c);maxPeers,e:=strconv.Atoi(env("CALL_MAX_ROOM_SIZE","8"));if e!=nil||maxPeers<2||maxPeers>32{maxPeers=8};if n>maxPeers{hub.leave(c);_ = conn.WriteJSON(map[string]string{"type":"error","error":"room_full"});conn.Close();return}
 defer func(){hub.leave(c);for _,p:=range hub.peers(c){send(p,map[string]any{"type":"peer-left","peer":c.user})};close(c.send);conn.Close()}();conn.SetReadDeadline(time.Time{})
 for _,p:=range hub.peers(c){send(c,map[string]any{"type":"peer-joined","peer":p.user})}
 go func(){for b:=range c.send{_ = conn.SetWriteDeadline(time.Now().Add(10*time.Second));if conn.WriteMessage(websocket.TextMessage,b)!=nil{return}}}()
 for{_,b,e:=conn.ReadMessage();if e!=nil{return};var msg map[string]any;if json.Unmarshal(b,&msg)!=nil{continue};t,_:=msg["type"].(string);if t!="offer"&&t!="answer"&&t!="ice"&&t!="hangup"&&t!="media-state"{continue};msg["from"]=c.user;to,_:=msg["to"].(string);for _,p:=range hub.peers(c){if to==""||to==p.user{send(p,msg)}};if t=="hangup"{return}}
}
func main(){if len(env("CALL_TOKEN_SECRET",""))<32{log.Fatal("CALL_TOKEN_SECRET must be at least 32 chars")};dsn:=fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&charset=utf8mb4",env("DB_USER","admin"),env("DB_PASSWORD",""),env("DB_HOST","localhost"),env("DB_PORT","3306"),env("DB_NAME","calls"));db,e:=sql.Open("mysql",dsn);if e!=nil{log.Fatal(e)};db.SetMaxOpenConns(20);if e=db.Ping();e!=nil{log.Fatal("database: ",e)};hub=&Hub{rooms:map[string]map[*Client]bool{},db:db};http.HandleFunc("/health",func(w http.ResponseWriter,r *http.Request){w.Header().Set("Content-Type","application/json");_ = json.NewEncoder(w).Encode(map[string]any{"ok":true,"service":"pluschat-calls","database":"connected","transport":"websocket"})});http.HandleFunc("/ws",wsHandler);addr:=env("ADDR","127.0.0.1:8090");log.Printf("listening on %s",addr);log.Fatal(http.ListenAndServe(addr,nil))}
